package websocket

import (
	"errors"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CheckinHandler struct {
	client *Client
}

func NewCheckinHandler(client *Client) *CheckinHandler {
	return &CheckinHandler{client: client}
}

var ErrUnexpectedResponse = errors.New("unexpected websocket response payload")

func (h *CheckinHandler) Handle() {
	if h == nil || h.client == nil {
		slog.Error("checkin handler: websocket client is nil")
		return
	}

	documentID, err := h.requestDocumentID()
	if err != nil {
		slog.Error("checkin handler: request document", "error", err)
		return
	}
	slog.Info("checkin handler: received document id", "document_id", documentID)

	// TODO: search booking by document_id.
	bookingFound := false
	if bookingFound {
		return
	}

	bookingID, err := h.requestBookingID()
	if err != nil {
		slog.Error("checkin handler: request booking id", "error", err)
		return
	}
	slog.Info("checkin handler: received booking id", "booking_id", bookingID)

	// TODO: search booking by booking_id.
	bookingFound = false
	if !bookingFound {
		slog.Error("checkin handler: booking not found")
	}
}

func (h *CheckinHandler) requestDocumentID() (string, error) {
	response, err := h.sendSystemRequest(dto.SystemRequest_REQUEST_DOCUMENT)
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetShowDocument() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetShowDocument().GetDocumentId(), nil
}

func (h *CheckinHandler) requestBookingID() (string, error) {
	response, err := h.sendSystemRequest(dto.SystemRequest_REQUEST_BOOKING_NUMBER)
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetShowBookingNumber() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetShowBookingNumber().GetBookingId(), nil
}

func (h *CheckinHandler) sendSystemRequest(request dto.SystemRequest) (*dto.ChatMessage, error) {
	requestID := uuid.NewString()
	message := &dto.ChatMessage{
		MessageId:       requestID,
		CorrelationId:   requestID,
		SentAt:          timestamppb.Now(),
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: "lodging.v1",
		Phase:           dto.LifecyclePhase_LIFECYCLE_PHASE_CHECK_IN,
		Payload: &dto.ChatMessage_SystemRequest{
			SystemRequest: request,
		},
	}

	return h.client.SendAndWait(message, 15*time.Second)
}
