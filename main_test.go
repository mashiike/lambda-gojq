package main

import (
	"context"
	"encoding/json"
	"testing"

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
	}
	h := lambda.NewHandler(handler)
	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {

			actual, err := h.Invoke(context.Background(), c.payload)
			require.NoError(t, err)
			require.JSONEq(t, string(c.expected), string(actual))
		})
	}
}
