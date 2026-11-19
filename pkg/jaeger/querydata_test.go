package jaeger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
	"github.com/grafana/grafana-jaeger-datasource/pkg/jaeger/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformDependenciesResponse(t *testing.T) {
	t.Run("simple_dependencies", func(t *testing.T) {
		dependencies := types.DependenciesResponse{
			Data: []types.ServiceDependency{
				{
					Parent:    "serviceA",
					Child:     "serviceB",
					CallCount: 1,
				},
				{
					Parent:    "serviceA",
					Child:     "serviceC",
					CallCount: 2,
				},
				{
					Parent:    "serviceB",
					Child:     "serviceC",
					CallCount: 3,
				},
			},
		}

		frames := transformDependenciesResponse(dependencies, "test")
		experimental.CheckGoldenJSONFrame(t, "./testdata", "simple_dependencies_nodes.golden", frames[0], false)
		experimental.CheckGoldenJSONFrame(t, "./testdata", "simple_dependencies_edges.golden", frames[1], false)
	})

	t.Run("empty_dependencies", func(t *testing.T) {
		dependencies := types.DependenciesResponse{
			Data: []types.ServiceDependency{},
		}

		frames := transformDependenciesResponse(dependencies, "test")
		experimental.CheckGoldenJSONFrame(t, "./testdata", "empty_dependencies_nodes.golden", frames[0], false)
		experimental.CheckGoldenJSONFrame(t, "./testdata", "empty_dependencies_edges.golden", frames[1], false)
	})

	t.Run("complex_dependencies", func(t *testing.T) {
		dependencies := types.DependenciesResponse{
			Data: []types.ServiceDependency{
				{
					Parent:    "frontend",
					Child:     "auth-service",
					CallCount: 150,
				},
				{
					Parent:    "frontend",
					Child:     "api-gateway",
					CallCount: 300,
				},
				{
					Parent:    "api-gateway",
					Child:     "user-service",
					CallCount: 200,
				},
				{
					Parent:    "api-gateway",
					Child:     "order-service",
					CallCount: 100,
				},
				{
					Parent:    "order-service",
					Child:     "payment-service",
					CallCount: 80,
				},
				{
					Parent:    "order-service",
					Child:     "inventory-service",
					CallCount: 90,
				},
				{
					Parent:    "user-service",
					Child:     "database",
					CallCount: 500,
				},
				{
					Parent:    "payment-service",
					Child:     "database",
					CallCount: 200,
				},
				{
					Parent:    "inventory-service",
					Child:     "database",
					CallCount: 300,
				},
			},
		}

		frames := transformDependenciesResponse(dependencies, "test")
		experimental.CheckGoldenJSONFrame(t, "./testdata", "complex_dependencies_nodes.golden", frames[0], false)
		experimental.CheckGoldenJSONFrame(t, "./testdata", "complex_dependencies_edges.golden", frames[1], false)
	})
}

func TestQueryDataSearchRouting(t *testing.T) {
	tests := []struct {
		name     string
		toggles  string
		wantPath string
	}{
		{name: "v1 API by default", toggles: "", wantPath: "/api/traces"},
		{name: "api/v3 with jaegerEnableGrpcEndpoint", toggles: "jaegerEnableGrpcEndpoint", wantPath: "/api/v3/traces"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				if strings.HasPrefix(r.URL.Path, "/api/v3/") {
					_, _ = w.Write([]byte(`{"result":{"resourceSpans":[]}}`))
					return
				}
				_, _ = w.Write([]byte(`{"data":[]}`))
			}))
			defer server.Close()
			client, err := New(server.Client(), log.NewNullLogger(), backend.DataSourceInstanceSettings{URL: server.URL, JSONData: []byte(`{}`)})
			require.NoError(t, err)

			ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: tt.toggles,
			}))
			resp, err := queryData(ctx, &DataSource{JaegerClient: client}, &backend.QueryDataRequest{
				Queries: []backend.DataQuery{{
					RefID:     "A",
					JSON:      []byte(`{"queryType":"search","service":"checkout"}`),
					TimeRange: backend.TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
				}},
			})
			require.NoError(t, err)
			require.NoError(t, resp.Responses["A"].Error)
			assert.Equal(t, tt.wantPath, gotPath)
		})
	}
}
