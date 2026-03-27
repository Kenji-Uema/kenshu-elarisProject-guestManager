package logging

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	base slog.Handler
}

func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.base.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{base: h.base.WithAttrs(attrs)}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{base: h.base.WithGroup(name)}
}

func NewLogger() *slog.Logger {
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	traceHandler := &TraceHandler{base: base}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return slog.New(traceHandler).With(
		"app", hostname,
	)
}

func GuestHTTPInfo(ctx context.Context, event string, attrs ...any) {
	slog.InfoContext(ctx, "guest http", append([]any{
		"component", "guest_http",
		"layer", "handler",
		"event", event,
	}, attrs...)...)
}

func GuestHTTPWarn(ctx context.Context, event string, attrs ...any) {
	slog.WarnContext(ctx, "guest http", append([]any{
		"component", "guest_http",
		"layer", "handler",
		"event", event,
	}, attrs...)...)
}

func GuestHTTPError(ctx context.Context, event string, attrs ...any) {
	slog.ErrorContext(ctx, "guest http", append([]any{
		"component", "guest_http",
		"layer", "handler",
		"event", event,
	}, attrs...)...)
}
