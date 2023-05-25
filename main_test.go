package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	cases := []struct {
		casename string
		payload  json.RawMessage
		expected json.RawMessage
	}{
		{
			casename: "count up 1st",
			payload:  json.RawMessage(`{"query":".count += 1 | if .wait < 1 then .wait=1 else .wait*=2 end"}`),
			expected: json.RawMessage(`{"count": 1, "wait":1}`),
		},
		{
			casename: "count up 2nd",
			payload:  json.RawMessage(`{"query":".count += 1 | if .wait < 1 then .wait=1 else .wait*=2 end","data":{"count": 1, "wait":1}}`),
			expected: json.RawMessage(`{"count": 2, "wait":2}`),
		},
		{
			casename: "count up 3rd",
			payload:  json.RawMessage(`{"query":".count += 1 | if .wait < 1 then .wait=1 else .wait*=2 end","data":{"count": 2, "wait":2}}`),
			expected: json.RawMessage(`{"count": 3, "wait":4}`),
		},
		{
			casename: "arrray map",
			payload:  json.RawMessage(`{"query":".data=[(.data[] |.fuga)]","data":{"data": [{"hoge":1,"fuga":2},{"hoge":2,"fuga":4}]}}`),
			expected: json.RawMessage(`{"data": [2,4]}`),
		},
	}
	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			h := lambda.NewHandler(newHandler("."))
			actual, err := h.Invoke(context.Background(), c.payload)
			require.NoError(t, err)
			require.JSONEq(t, string(c.expected), string(actual))
		})
	}
}

func TestFirehoseHandler__ForCloudwatchLogsSubscriptionFilter(t *testing.T) {
	data1 := base64.StdEncoding.EncodeToString([]byte(`{
		"messageType":"DATA_MESSAGE",
		"owner":"123456789012",
		"logGroup":"logGroup1",
		"logStream":"logStream1",
		"subscriptionFilters":[
			"filter1"
		],
		"logEvents":[
			{"id":"id1","timestamp":1495072949453,"message":"{\"hoge\":\"hoge1\",\"fuga\":\"fuga1\"}"},
			{"id":"id2","timestamp":1495072949477,"message":"{\"hoge\":\"hoge2\",\"fuga\":\"fuga2\"}"}
		]
	}` + "\n"))
	data2 := base64.StdEncoding.EncodeToString([]byte(`{
		"messageType":"DATA_MESSAGE",
		"owner":"123456789012",
		"logGroup":"logGroup2",
		"logStream":"logStream2",
		"subscriptionFilters":[
			"filter2"
		],
		"logEvents":[
			{"id":"id3","timestamp":1495072947777,"message":"{\"hoge\":\"hoge3\",\"fuga\":\"fuga3\"}"},
			{"id":"id4","timestamp":1495072949888,"message":"{\"hoge\":\"hoge4\",\"fuga\":\"fuga4\"}"}
		]
	}` + "\n"))
	paylaod := json.RawMessage(`{
		"invocationId": "invocationIdExample",
		"deliveryStreamArn": "arn:aws:kinesis:EXAMPLE",
		"region": "us-east-1",
		"records": [
			{
				"recordId": "49546986683135544286507457936321625675700192471156785154",
				"approximateArrivalTimestamp": 1495072949453,
				"data": "` + data1 + `"
			},
			{
				"recordId": "49546986683135544286507457936321625675700192471156785155",
				"approximateArrivalTimestamp": 1495072949453,
				"data": "` + data2 + `"
			}
		]
	}`)
	handler, err := newFirehoseHandler(".logEvents[] | .message | fromjson")
	require.NoError(t, err)
	h := lambda.NewHandler(handler)
	actual, err := h.Invoke(context.Background(), paylaod)
	require.NoError(t, err)
	var resp events.KinesisFirehoseResponse
	err = json.Unmarshal(actual, &resp)
	require.NoError(t, err)
	expected := [][]byte{
		[]byte(`{"hoge":"hoge1","fuga":"fuga1"}` + "\n" + `{"hoge":"hoge2","fuga":"fuga2"}` + "\n"),
		[]byte(`{"hoge":"hoge3","fuga":"fuga3"}` + "\n" + `{"hoge":"hoge4","fuga":"fuga4"}` + "\n"),
	}
	for i, r := range resp.Records {
		if r.Data == nil {
			continue
		}
		t.Log(string(r.Data))
		a := strings.Split(string(r.Data), "\n")
		b := strings.Split(string(expected[i]), "\n")
		if len(a) != len(b) {
			t.Errorf("len(a) != len(b) (%d != %d)", len(a), len(b))
		}
		for j := range a {
			if a[j] == "" {
				continue
			}
			require.JSONEq(t, b[j], a[j])
		}
	}
}
