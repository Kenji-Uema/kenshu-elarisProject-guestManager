package websocket

import (
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

func handleGuestAction(lodgingState *domain.LodgingState, action dto.GuestAction) {
	switch action {
	case dto.GuestAction_SHOW_FOR_CHECKIN:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_ENTER_COTTAGE:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_LEAVE_COTTAGE:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_GO_FOR_A_BATH:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_RETURN_FROM_BATH:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_GO_FOR_DINNER:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_GO_FOR_BREAKFAST:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_GO_TO_SLEEP:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_WAKEUP:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_ENJOY_RESORT:
		slog.Info("guest action", "action", action)
	case dto.GuestAction_PROCEED_TO_CHECKOUT:
		slog.Info("guest action", "action", action)
	default:
		slog.Error("unknown guest action")
	}
}
