package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/itchyny/gojq"
	"github.com/ken39arg/go-flagx"
	"golang.org/x/exp/slog"
)

var Version string = "current"

func main() {
	var (
		logLevel     string
		mode         string
		defaultQuery string
	)
	flag.StringVar(&logLevel, "log-level", "info", "output log level")
	flag.StringVar(&mode, "mode", "default", "handler mode(default|firehose)")
	flag.StringVar(&defaultQuery, "query", ".", "default query")
	flag.VisitAll(flagx.EnvToFlag)
	flag.Parse()
	var minLevel slog.Level
	var addSource bool
	switch {
	case strings.EqualFold(logLevel, "debug"):
		addSource = true
		minLevel = slog.LevelDebug
	case strings.EqualFold(logLevel, "info"):
		minLevel = slog.LevelInfo
	case strings.EqualFold(logLevel, "warn"):
		minLevel = slog.LevelWarn
	case strings.EqualFold(logLevel, "error"):
		minLevel = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.HandlerOptions{Level: minLevel, AddSource: addSource}.NewTextHandler(os.Stderr)))
	slog.Info("start up bootstarp", "version", Version)
	var handler interface{}
	switch mode {
	case "default":
		handler = newHandler(defaultQuery)
	case "firehose":
		var err error
		handler, err = newFirehoseHandler(defaultQuery)
		if err != nil {
			slog.Error("firehose handler init failed", "detail", err)
			os.Exit(1)
		}
	default:
		slog.Error("mode is unknown", "mode", mode)
		os.Exit(1)
	}

	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") || os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.StartWithOptions(handler)
	}
	h := lambda.NewHandler(handler)
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()
	output, err := h.Invoke(ctx, payload)
	if err != nil {
		slog.ErrorCtx(ctx, err.Error())
		os.Exit(1)
	}
	os.Stdout.Write(output)
	os.Stdout.Write([]byte("\n"))
}

type Payload struct {
	Query string      `json:"query"`
	Data  interface{} `json:"data"`
}

func newHandler(defaultQuery string) func(ctx context.Context, payload *Payload) (interface{}, error) {
	return func(ctx context.Context, payload *Payload) (interface{}, error) {
		slog.Info("handle invocation", "query", payload.Query, "version", Version)
		if payload.Query == "" {
			payload.Query = defaultQuery
		}
		query, err := gojq.Parse(payload.Query)
		if err != nil {
			slog.ErrorCtx(ctx, "query parse failed", "detail", err)
			return nil, err
		}

		output, err := runQuery(ctx, query, payload.Data)
		if err != nil {
			slog.ErrorCtx(ctx, "query run failed", "detail", err)
			return nil, err
		}
		return output, nil
	}
}

type firehoseHandlerFunc func(ctx context.Context, payload *events.KinesisFirehoseEvent) (*events.KinesisFirehoseResponse, error)

func newFirehoseHandler(rawQuery string) (firehoseHandlerFunc, error) {
	query, err := gojq.Parse(rawQuery)
	if err != nil {
		return nil, fmt.Errorf("query parse failed: %w", err)
	}
	h := func(ctx context.Context, payload *events.KinesisFirehoseEvent) (*events.KinesisFirehoseResponse, error) {
		resp := &events.KinesisFirehoseResponse{
			Records: make([]events.KinesisFirehoseResponseRecord, len(payload.Records)),
		}
		slog.InfoCtx(ctx, "handle invocation", "invocation_id", payload.InvocationID, "delivery_stream_arn", payload.DeliveryStreamArn, "records_count", len(payload.Records), "version", Version)
		var wg sync.WaitGroup
		for i, record := range payload.Records {
			wg.Add(1)
			go func(i int, record events.KinesisFirehoseEventRecord) {
				defer wg.Done()
				slog.DebugCtx(ctx, "record data dump", "record_id", record.RecordID, "data", string(record.Data))
				slog.InfoCtx(ctx, "handle record", "record_id", record.RecordID, "approximate_arrival_timestamp", record.ApproximateArrivalTimestamp)
				var data interface{}
				if err := json.Unmarshal(record.Data, &data); err != nil {
					slog.ErrorCtx(ctx, "record data unmarshal failed", "record_id", record.RecordID, "detail", err)
					resp.Records[i] = events.KinesisFirehoseResponseRecord{
						RecordID: record.RecordID,
						Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
					}
					return
				}
				output, err := runQuery(ctx, query, data)
				if err != nil {
					slog.ErrorCtx(ctx, "query run failed", "record_id", record.RecordID, "detail", err)
					resp.Records[i] = events.KinesisFirehoseResponseRecord{
						RecordID: record.RecordID,
						Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
					}
					return
				}
				if output == nil {
					resp.Records[i] = events.KinesisFirehoseResponseRecord{
						RecordID: record.RecordID,
						Result:   events.KinesisFirehoseTransformedStateDropped,
						Metadata: events.KinesisFirehoseResponseRecordMetadata{
							PartitionKeys: map[string]string{},
						},
					}
					return
				}

				b, err := json.Marshal(output)
				if err != nil {
					slog.ErrorCtx(ctx, "output marshal failed", "record_id", record.RecordID, "detail", err)
					resp.Records[i] = events.KinesisFirehoseResponseRecord{
						RecordID: record.RecordID,
						Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
						Metadata: events.KinesisFirehoseResponseRecordMetadata{
							PartitionKeys: map[string]string{},
						},
					}
					return
				}
				resp.Records[i] = events.KinesisFirehoseResponseRecord{
					RecordID: record.RecordID,
					Result:   events.KinesisFirehoseTransformedStateOk,
					Data:     b,
					Metadata: events.KinesisFirehoseResponseRecordMetadata{
						PartitionKeys: map[string]string{},
					},
				}
			}(i, record)
		}
		wg.Wait()
		return resp, nil
	}
	return h, nil
}

func runQuery(ctx context.Context, query *gojq.Query, data interface{}) (interface{}, error) {
	iter := query.RunWithContext(ctx, data)
	output := make([]interface{}, 0)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("query iter err: %w", err)
		}
		output = append(output, v)
	}
	if len(output) == 0 {
		return nil, nil
	}
	if len(output) == 1 {
		return output[0], nil
	}
	return output, nil
}
