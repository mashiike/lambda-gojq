package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/itchyny/gojq"
	"github.com/ken39arg/go-flagx"
	"golang.org/x/exp/slog"
)

var Version string = "current"

func main() {
	var (
		logLevel string
	)
	flag.StringVar(&logLevel, "log-level", "info", "output log level")
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

func handler(ctx context.Context, payload *Payload) (interface{}, error) {
	slog.Info("handle invocation", "query", payload.Query, "version", Version)
	query, err := gojq.Parse(payload.Query)
	if err != nil {
		slog.ErrorCtx(ctx, "query parse failed", "detail", err)
		return nil, err
	}

	iter := query.RunWithContext(ctx, payload.Data)
	output := make([]interface{}, 0)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Printf("[error] query iter err: %s", err.Error())
			return nil, err
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
