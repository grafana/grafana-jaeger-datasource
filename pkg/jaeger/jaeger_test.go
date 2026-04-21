package jaeger

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-jaeger-datasource/pkg/jaeger/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataSourceInstanceSettings_TraceIdTimeEnabled(t *testing.T) {
	tests := []struct {
		name            string
		jsonData        string
		expectedEnabled bool
		expectError     bool
	}{
		{
			name: "traceIdTimeParams enabled",
			jsonData: `{
				"traceIdTimeParams": {
					"enabled": true
				}
			}`,
			expectedEnabled: true,
			expectError:     false,
		},
		{
			name: "traceIdTimeParams disabled",
			jsonData: `{
				"traceIdTimeParams": {
					"enabled": false
				}
			}`,
			expectedEnabled: false,
			expectError:     false,
		},
		{
			name:            "traceIdTimeParams not specified",
			jsonData:        `{}`,
			expectedEnabled: false,
			expectError:     false,
		},
		{
			name:            "traceIdTimeParams without enabled",
			jsonData:        `{"traceIdTimeParams":{}}`,
			expectedEnabled: false,
			expectError:     false,
		},
		{
			name:            "Invalid JSON",
			jsonData:        `{invalid json`,
			expectedEnabled: false,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := backend.DataSourceInstanceSettings{
				JSONData: []byte(tt.jsonData),
				URL:      "http://localhost:16686",
			}

			instance, err := NewDatasource(context.Background(), settings)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, instance)

			ds, ok := instance.(*DataSource)
			require.True(t, ok)
			require.NotNil(t, ds)

			var jsonData types.SettingsJSONData
			if err := json.Unmarshal(ds.JaegerClient.settings.JSONData, &jsonData); err != nil {
				t.Fatalf("failed to parse settings JSON data: %v", err)
			}

			assert.Equal(t, tt.expectedEnabled, jsonData.TraceIdTimeParams.Enabled)
		})
	}
}
