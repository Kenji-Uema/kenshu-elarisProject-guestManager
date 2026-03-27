package handler

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

func logHandlerInfo(ctx context.Context, event string, attrs ...any) {
	slog.InfoContext(ctx, "lodging chat", append(baseHandlerAttrs(event), attrs...)...)
}

func logHandlerWarn(ctx context.Context, event string, attrs ...any) {
	slog.WarnContext(ctx, "lodging chat", append(baseHandlerAttrs(event), attrs...)...)
}

func logHandlerError(ctx context.Context, event string, attrs ...any) {
	slog.ErrorContext(ctx, "lodging chat", append(baseHandlerAttrs(event), attrs...)...)
}

func baseHandlerAttrs(event string) []any {
	return []any{
		"component", "lodging_chat",
		"layer", "handler",
		"event", event,
	}
}

func guestActionAttrs(msg *dto.ChatMessage) []any {
	return []any{
		"action", msg.GetGuestAction().String(),
		"message_id", msg.GetMessageId(),
		"correlation_id", msg.GetCorrelationId(),
	}
}
