package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	data1 := base64.StdEncoding.EncodeToString([]byte(`{"id":"id1","timestamp":1495072949453,"message":"{\"logGroup\":\"logGroup1\",\"logStream\":\"logStream1\",\"subscriptionFilters\":[\"subscriptionFilter1\",\"subscriptionFilter2\"]}"}`))
	data2 := base64.StdEncoding.EncodeToString([]byte(`{"id":"id2","timestamp":1495072949453,"message":"{\"logGroup\":\"logGroup2\",\"logStream\":\"logStream2\",\"subscriptionFilters\":[\"subscriptionFilter1\",\"subscriptionFilter2\"]}"}`))
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
	result1 := base64.StdEncoding.EncodeToString([]byte(`{"logGroup":"logGroup1","logStream":"logStream1","subscriptionFilters":["subscriptionFilter1","subscriptionFilter2"]}`))
	result2 := base64.StdEncoding.EncodeToString([]byte(`{"logGroup":"logGroup2","logStream":"logStream2","subscriptionFilters":["subscriptionFilter1","subscriptionFilter2"]}`))
	expected := json.RawMessage(`{
		"records": [
			{
				"recordId": "49546986683135544286507457936321625675700192471156785154",
				"result": "Ok",
				"data": "` + result1 + `",
				"metadata": {
					"partitionKeys": {}
				}
			},
			{
				"recordId": "49546986683135544286507457936321625675700192471156785155",
				"result": "Ok",
				"data": "` + result2 + `",
				"metadata": {
					"partitionKeys": {}
				}
			}
		]
	}`)
	handler, err := newFirehoseHandler(".message | fromjson")
	require.NoError(t, err)
	h := lambda.NewHandler(handler)
	actual, err := h.Invoke(context.Background(), paylaod)
	require.NoError(t, err)
	var resp events.KinesisFirehoseResponse
	err = json.Unmarshal(actual, &resp)
	require.NoError(t, err)
	for _, r := range resp.Records {
		if r.Data == nil {
			continue
		}
		t.Log(string(r.Data))
	}
	require.JSONEq(t, string(expected), string(actual))
}
