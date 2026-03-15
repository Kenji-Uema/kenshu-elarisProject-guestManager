package handler

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

func handleGuestAction(lodgingState *domain.LodgingState, action dto.GuestAction) {
	ctx := context.Background()

	switch action {
	case dto.GuestAction_SHOW_FOR_CHECKIN:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_ENTER_COTTAGE:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_LEAVE_COTTAGE:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_GO_FOR_A_BATH:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_RETURN_FROM_BATH:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_GO_FOR_DINNER:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_GO_FOR_BREAKFAST:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_GO_TO_SLEEP:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_WAKEUP:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_ENJOY_RESORT:
		slog.InfoContext(ctx, "guest action", "action", action)
	case dto.GuestAction_PROCEED_TO_CHECKOUT:
		slog.InfoContext(ctx, "guest action", "action", action)
	default:
		slog.ErrorContext(ctx, "unknown guest action")
	}
}
