package handler

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/transport/grpc/clock"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/message"
)

type CheckoutHandler struct {
	receptionService app.ReceptionService
	clockEmu         clock.Emu
	writer           *message.Writer
	reader           *message.Reader
}

func NewCheckoutHandler(receptionService app.ReceptionService, writer *message.Writer, reader *message.Reader) *CheckoutHandler {
	return &CheckoutHandler{receptionService: receptionService, writer: writer, reader: reader}
}

func (h CheckoutHandler) ProcessCheckout(ctx context.Context) error {
	now, err := h.clockEmu.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "checkin handler: clock emu now", "error", err)
		return err
	}

	_, err = h.reader.WaitForGuestAction(ctx, dto.GuestAction_PROCEED_TO_CHECKOUT)
	if err != nil {
		slog.WarnContext(ctx, "wait for SHOW_FOR_CHECKIN", "error", err)
		return err
	}

	if _, err := h.requestCottageKey(ctx); err != nil {
		return err
	}

	if err := h.receptionService.CheckOut(ctx, "", *now); err != nil {
		return err
	}

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_IN_COMPLETE); err != nil {
		return err
	}

	return nil
}

func (h CheckoutHandler) requestCottageKey(ctx context.Context) (string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_COTTAGE_KEY, dto.GuestAction_RETURN_COTTAGE_KEY)
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetReturnCottageKey() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetReturnCottageKey().GetCottageKeyId(), nil
}
