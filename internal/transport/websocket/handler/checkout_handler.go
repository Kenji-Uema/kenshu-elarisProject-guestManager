package handler

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
)

type CheckoutHandler struct {
	receptionService app.ReceptionService
	clock            port.ClockClient
	writer           chat.Writer
	reader           chat.Reader
}

func NewCheckoutHandler(receptionService app.ReceptionService, clock port.ClockClient, writer chat.Writer, reader chat.Reader) *CheckoutHandler {
	return &CheckoutHandler{receptionService: receptionService, clock: clock, writer: writer, reader: reader}
}

func (h CheckoutHandler) Handle(ctx context.Context, booking domain.Booking) error {
	now, err := h.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "checkin handler: clock emu now", "error", err)
		return err
	}

	_, err = h.reader.WaitForGuestAction(ctx, dto.GuestAction_PROCEED_TO_CHECKOUT)
	if err != nil {
		slog.WarnContext(ctx, "wait for SHOW_FOR_CHECKIN", "error", err)
		return err
	}

	keyNumber, err := h.requestCottageKey(ctx)
	if err != nil {
		return err
	}
	if _, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_RETURN_COTTAGE_KEY); err != nil {
		return err
	}
	if err := h.receptionService.ReturnCottageKey(ctx, booking.CottageName, keyNumber); err != nil {
		return err
	}

	if err := h.receptionService.CheckOut(ctx, booking, *now); err != nil {
		return err
	}

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_OUT_COMPLETE); err != nil {
		return err
	}

	return nil
}

func (h CheckoutHandler) requestCottageKey(ctx context.Context) (string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_COTTAGE_KEY, &dto.GuestResponse{
		Payload: &dto.GuestResponse_ReturnCottageKey{},
	})
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetReturnCottageKey() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetReturnCottageKey().GetCottageKeyId(), nil
}
