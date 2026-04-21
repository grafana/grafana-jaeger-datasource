package jaeger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

var logger = backend.NewLoggerWith("logger", "tsdb.jaeger")

type DataSource struct {
	JaegerClient JaegerClient
}

type datasourceJSONData struct {
	TraceIdTimeParams struct {
		Enabled bool `json:"enabled"`
	} `json:"traceIdTimeParams"`
}

func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	httpClientOptions, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, backend.DownstreamError(fmt.Errorf("error reading settings: %w", err))
	}

	httpClient, err := httpclient.NewProvider().New(httpClientOptions)
	if err != nil {
		return nil, fmt.Errorf("error creating http client: %w", err)
	}

	if settings.URL == "" {
		return nil, backend.DownstreamError(errors.New("error reading settings: url is empty"))
	}

	var jsonData datasourceJSONData
	err = json.Unmarshal(settings.JSONData, &jsonData)
	if err != nil {
		return nil, fmt.Errorf("error reading settings: %w", err)
	}

	logger := logger.FromContext(ctx)
	jaegerClient, err := New(httpClient, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("error creating jaeger client: %w", err)
	}

	return &DataSource{JaegerClient: jaegerClient}, err
}

func (ds *DataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	cfg := backend.GrafanaConfigFromContext(ctx)

	var servicesErr error
	if cfg.FeatureToggles().IsEnabled("jaegerEnableGrpcEndpoint") {
		_, servicesErr = ds.JaegerClient.GrpcServices(ctx)
	} else {
		_, servicesErr = ds.JaegerClient.Services(ctx)
	}

	if servicesErr != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: servicesErr.Error(),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

func (ds *DataSource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	handler := httpadapter.New(ds.registerResourceRoutes())
	return handler.CallResource(ctx, req, sender)
}

func (ds *DataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return queryData(ctx, ds, req)
}
