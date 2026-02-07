//package probe
//
//import (
//	"context"
//	"fmt"
//	"io"
//	"log/slog"
//	"net/http"
//
//	"github.com/Kenji-Uema/clockEmulator/internal/config"
//	"github.com/Kenji-Uema/clockEmulator/internal/infra/mq"
//)
//
//func ReadinessHandler(rabbitMqClient *mq.RabbitMqConnection, telemetryCfg config.TelemetryConfig) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		if !rabbitMqClient.IsConnectionOpen() {
//			http.Error(w, "rabbitmq connection closed", http.StatusServiceUnavailable)
//			return
//		}
//		if err := checkOtelCollector(r.Context(), telemetryCfg); err != nil {
//			http.Error(w, "otel collector unreachable", http.StatusServiceUnavailable)
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//	}
//}
//
//func checkOtelCollector(ctx context.Context, cfg config.TelemetryConfig) error {
//	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s:%d/health", cfg.OTLPEndpoint, cfg.OTLPHealthPort), nil)
//	if err != nil {
//		return err
//	}
//
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		return err
//	}
//	defer func(Body io.ReadCloser) {
//		err := Body.Close()
//		if err != nil {
//			slog.Error("close response body", "error", err)
//		}
//	}(resp.Body)
//
//	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
//		return fmt.Errorf("otel collector health check failed: status %d", resp.StatusCode)
//	}
//	return nil
//}
