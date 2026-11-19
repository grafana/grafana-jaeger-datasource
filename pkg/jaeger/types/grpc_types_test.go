package types

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcDoubleUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{name: "number", input: `3.5`, want: 3.5},
		{name: "zero", input: `0`, want: 0},
		{name: "quoted number", input: `"2.25"`, want: 2.25},
		{name: "Infinity", input: `"Infinity"`, want: math.Inf(1)},
		{name: "-Infinity", input: `"-Infinity"`, want: math.Inf(-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d GrpcDouble
			require.NoError(t, json.Unmarshal([]byte(tt.input), &d))
			assert.Equal(t, tt.want, float64(d))
		})
	}

	t.Run("NaN", func(t *testing.T) {
		var d GrpcDouble
		require.NoError(t, json.Unmarshal([]byte(`"NaN"`), &d))
		assert.True(t, math.IsNaN(float64(d)))
	})

	t.Run("invalid", func(t *testing.T) {
		var d GrpcDouble
		assert.Error(t, json.Unmarshal([]byte(`"not-a-number"`), &d))
		assert.Error(t, json.Unmarshal([]byte(`true`), &d))
	})
}

// The fixture is a trace returned by Jaeger 2.21's /api/v3/traces/{id}, with one
// attribute of each protobuf JSON type.
func TestGrpcTracesResponseDecodesJaegerAttributeTypes(t *testing.T) {
	raw, err := os.ReadFile("../testdata/grpc_trace_attribute_types.json")
	require.NoError(t, err)

	var response GrpcTracesResponse
	require.NoError(t, json.Unmarshal(raw, &response))

	values := map[string]GrpcAnyValue{}
	for _, rs := range response.Result.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if span.Name == "root-op" {
					for _, a := range span.Attributes {
						values[a.Key] = a.Value
					}
				}
			}
		}
	}

	require.NotNil(t, values["bool.true"].BoolValue)
	assert.True(t, *values["bool.true"].BoolValue)
	require.NotNil(t, values["bool.false"].BoolValue)
	assert.False(t, *values["bool.false"].BoolValue)
	require.NotNil(t, values["double.attr"].DoubleValue)
	assert.Equal(t, 3.5, float64(*values["double.attr"].DoubleValue))
	require.NotNil(t, values["double.zero"].DoubleValue)
	assert.Equal(t, 0.0, float64(*values["double.zero"].DoubleValue))
	assert.Equal(t, "42", values["int.attr"].IntValue)
	assert.Equal(t, "hello", values["str.attr"].StringValue)
	require.Len(t, values["array.attr"].ArrayValue.Values, 2)
	assert.Equal(t, "a", values["array.attr"].ArrayValue.Values[0].StringValue)
	assert.Equal(t, "2", values["array.attr"].ArrayValue.Values[1].IntValue)
}
