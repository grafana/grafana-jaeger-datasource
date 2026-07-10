package jaeger

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

func (ds *DataSource) registerResourceRoutes() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("GET /services", ds.getServicesHandler())
	router.HandleFunc("GET /services/{service}/operations", ds.getOperationsHandler())
	return router
}

func (ds *DataSource) getServicesHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := config.GrafanaConfigFromContext(ctx)
		var services []string
		var err error
		if cfg.FeatureToggles().IsEnabled("jaegerEnableGrpcEndpoint") {
			services, err = ds.JaegerClient.GrpcServices(ctx)
		} else {
			services, err = ds.JaegerClient.Services(ctx)
		}
		writeResponse(services, err, rw, ds.JaegerClient.logger)
	}
}

func (ds *DataSource) getOperationsHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := config.GrafanaConfigFromContext(ctx)
		service := strings.TrimSpace(r.PathValue("service"))
		var operations []string
		var err error
		if cfg.FeatureToggles().IsEnabled("jaegerEnableGrpcEndpoint") {
			operations, err = ds.JaegerClient.GrpcOperations(ctx, service)
		} else {
			operations, err = ds.JaegerClient.Operations(ctx, service)
		}
		writeResponse(operations, err, rw, ds.JaegerClient.logger)
	}
}

func writeResponse(res interface{}, err error, rw http.ResponseWriter, logger log.Logger) {
	if err != nil {
		// This is used for resource calls, we don't need to add actual error message, but we should log it
		logger.Warn("An error occurred while doing a resource call", "error", err)
		http.Error(rw, "An error occurred within the plugin", http.StatusInternalServerError)
		return
	}
	// Response should not be string, but just in case, handle it
	if str, ok := res.(string); ok {
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte(str))
		return
	}
	b, err := json.Marshal(res)
	if err != nil {
		// This is used for resource calls, we don't need to add actual error message, but we should log it
		logger.Warn("An error occurred while processing response from resource call", "error", err)
		http.Error(rw, "An error occurred within the plugin", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(b)
}
