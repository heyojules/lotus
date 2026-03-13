package otlpreceiver

import (
	"context"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"

	"github.com/tinytelemetry/tiny-telemetry/internal/ingest"
	"github.com/tinytelemetry/tiny-telemetry/internal/model"
)

// logsHandler implements the OTLP LogsService gRPC server.
type logsHandler struct {
	collogspb.UnimplementedLogsServiceServer
	sink model.RecordSink
}

// Export handles an incoming ExportLogsServiceRequest.
func (h *logsHandler) Export(ctx context.Context, req *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	for _, rl := range req.GetResourceLogs() {
		resourceAttrs := extractResourceAttrs(rl.GetResource())

		for _, sl := range rl.GetScopeLogs() {
			scopeAttrs := ingest.CloneAttributes(resourceAttrs)

			if scope := sl.GetScope(); scope != nil {
				if scope.Name != "" {
					scopeAttrs["otel.scope.name"] = scope.Name
				}
				if scope.Version != "" {
					scopeAttrs["otel.scope.version"] = scope.Version
				}
				mergeKeyValues(scopeAttrs, scope.Attributes)
			}

			for _, lr := range sl.GetLogRecords() {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				record := convertLogRecord(lr, scopeAttrs)
				h.sink.Add(record)
			}
		}
	}

	return &collogspb.ExportLogsServiceResponse{}, nil
}
