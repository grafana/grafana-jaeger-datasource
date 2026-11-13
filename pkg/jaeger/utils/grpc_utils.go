package utils

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-jaeger-datasource/pkg/jaeger/types"
)

func TransformGrpcSearchResponse(response types.GrpcTracesResult, dsUID string, dsName string, limit int) *data.Frame {
	// Create a frame for the traces
	frame := data.NewFrame("traces",
		data.NewField("traceID", nil, []string{}).SetConfig(&data.FieldConfig{
			DisplayName: "Trace ID",
			Links: []data.DataLink{
				{
					Title: "Trace: ${__value.raw}",
					URL:   "",
					Internal: &data.InternalDataLink{
						DatasourceUID:  dsUID,
						DatasourceName: dsName,
						Query: map[string]interface{}{
							"query": "${__value.raw}",
						},
					},
				},
			},
		}),
		data.NewField("traceName", nil, []string{}).SetConfig(&data.FieldConfig{
			DisplayName: "Trace name",
		}),
		data.NewField("startTime", nil, []time.Time{}).SetConfig(&data.FieldConfig{
			DisplayName: "Start time",
		}),
		data.NewField("duration", nil, []int64{}).SetConfig(&data.FieldConfig{
			DisplayName: "Duration",
			Unit:        "µs",
		}),
	)

	// Set the visualization type to table
	frame.Meta = &data.FrameMeta{
		PreferredVisualization: "table",
	}

	// A trace's spans can arrive in several resourceSpans (one per service), so group
	// them by trace ID and take the earliest span as the root, as the v1 search does.
	type traceRow struct {
		traceID     string
		serviceName string
		root        types.GrpcSpan
		rootStart   int64
	}
	rowsByID := map[string]*traceRow{}
	for _, res := range response.ResourceSpans {
		serviceName := getAttribute(res.Resource.Attributes, "service.name").StringValue
		for _, scopeSpan := range res.ScopeSpans {
			for _, span := range scopeSpan.Spans {
				start, err := strconv.ParseInt(span.StartTimeUnixNano, 10, 64)
				if err != nil {
					continue
				}
				row, ok := rowsByID[span.TraceID]
				if !ok || start < row.rootStart {
					rowsByID[span.TraceID] = &traceRow{traceID: span.TraceID, serviceName: serviceName, root: span, rootStart: start}
				}
			}
		}
	}

	rows := make([]*traceRow, 0, len(rowsByID))
	for _, row := range rowsByID {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].rootStart != rows[j].rootStart {
			return rows[i].rootStart > rows[j].rootStart
		}
		return rows[i].traceID < rows[j].traceID
	})

	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}

	for _, row := range rows {
		duration := int64(0)
		if end, err := strconv.ParseInt(row.root.EndTimeUnixNano, 10, 64); err == nil {
			duration = (end - row.rootStart) / 1000 // microseconds
		}
		frame.AppendRow(
			row.traceID,
			fmt.Sprintf("%s: %s", row.serviceName, row.root.Name),
			time.Unix(0, row.rootStart),
			duration,
		)
	}

	return frame
}

func TransformGrpcTraceResponse(trace []types.GrpcResourceSpans, refID string) *data.Frame {
	frame := data.NewFrame(refID,
		data.NewField("traceID", nil, []string{}),
		data.NewField("spanID", nil, []string{}),
		data.NewField("parentSpanID", nil, []string{}),
		data.NewField("statusCode", nil, []int64{}),
		data.NewField("statusMessage", nil, []string{}),
		data.NewField("kind", nil, []string{}),
		data.NewField("operationName", nil, []string{}),
		data.NewField("serviceName", nil, []string{}),
		data.NewField("serviceTags", nil, []json.RawMessage{}),
		data.NewField("startTime", nil, []float64{}),
		data.NewField("duration", nil, []float64{}),
		data.NewField("logs", nil, []json.RawMessage{}),
		data.NewField("references", nil, []json.RawMessage{}),
		data.NewField("tags", nil, []json.RawMessage{}),
	)

	// Set metadata for trace visualization
	frame.Meta = &data.FrameMeta{
		PreferredVisualization: "trace",
		Custom: map[string]interface{}{
			"traceFormat": "jaeger",
		},
	}

	// each resource is a difference service name or "process"
	for _, resource := range trace {
		for _, scopeSpan := range resource.ScopeSpans {
			for _, span := range scopeSpan.Spans {
				parentSpanID := span.ParentSpanID
				// Get service name and tags
				serviceName := getAttribute(resource.Resource.Attributes, "service.name").StringValue
				serviceTags := json.RawMessage{}
				processedResAttributes := processAttributes(resource.Resource.Attributes)
				tagsMarshaled, err := json.Marshal(processedResAttributes)
				if err == nil {
					serviceTags = json.RawMessage(tagsMarshaled)
				}

				// Convert tags
				tags := json.RawMessage{}
				processedSpanAttributes := processAttributes(span.Attributes)
				// add otel attributes scope name, scope version and span kind
				if scopeSpan.Scope.Name != "" {
					processedSpanAttributes = append(processedSpanAttributes, types.KeyValueType{
						Key:   "otel.scope.name",
						Value: scopeSpan.Scope.Name,
						Type:  "string",
					})
				}

				if scopeSpan.Scope.Version != "" {
					processedSpanAttributes = append(processedSpanAttributes, types.KeyValueType{
						Key:   "otel.scope.version",
						Value: scopeSpan.Scope.Version,
						Type:  "string",
					})
				}

				tagsMarshaled, err = json.Marshal(processedSpanAttributes)
				if err == nil {
					tags = json.RawMessage(tagsMarshaled)
				}

				// Convert logs
				// In the new API (OTLP based), logs are span events. See:
				// https://github.com/jaegertracing/jaeger-idl/blob/7c7460fc400325ae69435c0aa65697f4cc1ab581/swagger/api_v3/query_service.swagger.json#L630C9-L636C11
				logs := json.RawMessage{}
				processedEvents := convertGrpcEventsToLogs(span.Events)
				logsMarshaled, err := json.Marshal(processedEvents)
				if err == nil {
					logs = json.RawMessage(logsMarshaled)
				}

				// Convert references (excluding parent)
				references := json.RawMessage{}
				filteredLinks := []types.GrpcSpanLink{}
				// in the new API (OTLP based), references are defined as "SpanLinks" see:
				// https://github.com/jaegertracing/jaeger-idl/blob/7c7460fc400325ae69435c0aa65697f4cc1ab581/swagger/api_v3/query_service.swagger.json#L642C8-L648C11
				for _, ref := range span.Links {
					if parentSpanID == "" || ref.SpanID != parentSpanID {
						filteredLinks = append(filteredLinks, ref)
					}
				}
				processedLinks := convertGrpcLinkToReference(filteredLinks)
				refsMarshaled, err := json.Marshal(processedLinks)
				if err == nil {
					references = json.RawMessage(refsMarshaled)
				}

				// convert start time and calculate duration
				startTimeFloat, startErr := strconv.ParseFloat(span.StartTimeUnixNano, 64)
				endTimeFloat, endErr := strconv.ParseFloat(span.EndTimeUnixNano, 64)
				duration := float64(0)
				if startErr == nil && endErr == nil {
					duration = (endTimeFloat - startTimeFloat) / 1000000 // convert to milliseconds
				}

				// Add span to frame
				frame.AppendRow(
					span.TraceID,
					span.SpanID,
					parentSpanID,
					span.Status.Code,
					span.Status.Message,
					processSpanKind(span.Kind),
					span.Name,
					serviceName,
					serviceTags,
					startTimeFloat/1000000, // Convert nanoseconds to milliseconds
					duration,
					logs,
					references,
					tags,
				)
			}
		}
	}

	return frame
}

func processAttributes(attributes []types.GrpcKeyValue) []types.KeyValueType {
	tags := []types.KeyValueType{}

	for _, att := range attributes {
		if att.Value.StringValue != "" {
			tags = append(tags, types.KeyValueType{
				Key:   att.Key,
				Value: att.Value.StringValue,
				Type:  "string",
			})
			continue
		}

		if att.Value.BoolValue != nil {
			tags = append(tags, types.KeyValueType{
				Key:   att.Key,
				Value: *att.Value.BoolValue,
				Type:  "boolean",
			})
			continue
		}

		if att.Value.IntValue != "" {
			intVal, err := strconv.Atoi(att.Value.IntValue)
			if err != nil {
				continue
			}
			tags = append(tags, types.KeyValueType{
				Key:   att.Key,
				Value: int64(intVal),
				Type:  "int64",
			})
			continue
		}

		if att.Value.DoubleValue != nil {
			tags = append(tags, types.KeyValueType{
				Key:   att.Key,
				Value: float64(*att.Value.DoubleValue),
				Type:  "float64",
			})
			continue
		}

		// Arrays and maps are shown as JSON strings, as the v1 API returns them
		if len(att.Value.ArrayValue.Values) > 0 || len(att.Value.KvListValue.Values) > 0 {
			encoded, err := json.Marshal(plainValue(att.Value))
			if err != nil {
				continue
			}
			tags = append(tags, types.KeyValueType{
				Key:   att.Key,
				Value: string(encoded),
				Type:  "string",
			})
			continue
		}

		if att.Value.BytesValue != "" {
			tags = append(tags, types.KeyValueType{
				Key:   att.Key,
				Value: att.Value.BytesValue,
				Type:  "bytes",
			})
			continue
		}
	}
	return tags
}

func plainValue(v types.GrpcAnyValue) interface{} {
	switch {
	case v.BoolValue != nil:
		return *v.BoolValue
	case v.DoubleValue != nil:
		return float64(*v.DoubleValue)
	case v.IntValue != "":
		if i, err := strconv.ParseInt(v.IntValue, 10, 64); err == nil {
			return i
		}
		return v.IntValue
	case len(v.ArrayValue.Values) > 0:
		values := make([]interface{}, 0, len(v.ArrayValue.Values))
		for _, item := range v.ArrayValue.Values {
			values = append(values, plainValue(item))
		}
		return values
	case len(v.KvListValue.Values) > 0:
		values := make(map[string]interface{}, len(v.KvListValue.Values))
		for _, kv := range v.KvListValue.Values {
			values[kv.Key] = plainValue(kv.Value)
		}
		return values
	case v.BytesValue != "":
		return v.BytesValue
	default:
		return v.StringValue
	}
}

func getAttribute(attributes []types.GrpcKeyValue, attName string) types.GrpcAnyValue {
	var attValue types.GrpcAnyValue
	for _, att := range attributes {
		if att.Key == attName {
			return att.Value
		}
	}

	return attValue
}

func processSpanKind(kind int64) string {
	switch kind {
	case 0:
		return "unspecified"
	case 1:
		return "internal"
	case 2:
		return "server"
	case 3:
		return "client"
	case 4:
		return "producer"
	case 5:
		return "consumer"
	default:
		return "unspecified"
	}
}

// This is to help ensure backwards compatibility with the current non OTLP based Jager trace format
// a few fields are different between TraceLogs and GrpcSpanEvents
func convertGrpcEventsToLogs(events []types.GrpcSpanEvent) []types.TraceLog {
	logs := make([]types.TraceLog, 0, len(events))

	for _, event := range events {
		timestampNano, err := strconv.Atoi(event.TimeUnixNano)
		var timestampMs float64
		if err == nil {
			timestampMs = float64(timestampNano) / 1000000.0 // converting from nanoseconds to milliseconds
		}
		log := types.TraceLog{
			Name:      event.Name,
			Timestamp: timestampMs,
			Fields:    processAttributes(event.Attributes),
		}
		logs = append(logs, log)
	}

	return logs
}

// this is to help ensure backwards compatibility between references and links with the current non OTLP based Jaeger trace format
// There is no concept of RefType in the new OTLP based SpanLink, so we are only converting the SpanID and TraceID
func convertGrpcLinkToReference(links []types.GrpcSpanLink) []types.TraceSpanReference {
	references := make([]types.TraceSpanReference, 0, len(links))

	for _, ref := range links {
		references = append(references, types.TraceSpanReference{
			TraceID: ref.TraceID,
			SpanID:  ref.SpanID,
		})
	}

	return references
}
