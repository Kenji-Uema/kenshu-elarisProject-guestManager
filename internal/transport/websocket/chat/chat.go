package chat

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/chatErrors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

var ErrEmptyMessageID = errors.New("chat message must include message_id")

type Chat interface {
	ReadOneMessage(ctx context.Context) (*dto.ChatMessage, error)
	ReadAck(ctx context.Context) (*dto.ChatMessage, error)
	ReadReply(ctx context.Context) (*dto.ChatMessage, error)
	SendAck(ctx context.Context, msg *dto.ChatMessage)
	WriteOneMessage(ctx context.Context, msg *dto.ChatMessage) error
}

type conn interface {
	SetWriteDeadline(t time.Time) error
	WriteMessage(messageType int, data []byte) error
	SetReadDeadline(t time.Time) error
	ReadMessage() (messageType int, p []byte, err error)
}

type chat struct {
	conn         conn
	replyTimeout time.Duration
	ackTimeout   time.Duration
	writeMu      sync.Mutex
}

func NewChat(conn *websocket.Conn, replyTimeout time.Duration, ackTimeout time.Duration) (Reader, Writer) {
	chat := &chat{
		conn:         conn,
		replyTimeout: replyTimeout,
		ackTimeout:   ackTimeout,
	}

	return &reader{chat: chat}, &writer{chat: chat}
}

func (m *chat) WriteOneMessage(ctx context.Context, msg *dto.ChatMessage) error {
	if msg.GetMessageId() == "" {
		return ErrEmptyMessageID
	}

	b, err := protojson.Marshal(msg)
	if err != nil {
		slog.WarnContext(ctx, "lodging chat",
			"component", "lodging_chat",
			"layer", "transport",
			"event", "message_marshal_failed",
			"error", err,
			"message_id", msg.GetMessageId(),
		)
		return err
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if err := m.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	if err := m.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return err
	}

	slog.InfoContext(ctx, "lodging chat", outgoingMessageAttrs(msg)...)

	return nil
}

func (m *chat) ReadOneMessage(ctx context.Context) (*dto.ChatMessage, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := m.conn.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
	} else {
		if err := m.conn.SetReadDeadline(time.Time{}); err != nil {
			return nil, err
		}
	}

	_, b, err := m.conn.ReadMessage()

	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			slog.WarnContext(ctx, "lodging chat",
				"component", "lodging_chat",
				"layer", "transport",
				"event", "message_read_failed",
				"error", err,
			)
		}
		return nil, err
	}

	var msg dto.ChatMessage
	if err := protojson.Unmarshal(b, &msg); err != nil {
		slog.WarnContext(ctx, "lodging chat",
			"component", "lodging_chat",
			"layer", "transport",
			"event", "message_unmarshal_failed",
			"error", err,
		)
		return nil, err
	}

	slog.InfoContext(ctx, "lodging chat", incomingMessageAttrs(&msg)...)

	return &msg, nil
}

func (m *chat) ReadAck(ctx context.Context) (*dto.ChatMessage, error) {
	return m.readWithTimeout(ctx, m.ackTimeout, func(err error) error {
		return &chatErrors.AckNotReceivedErr{Err: err}
	})
}

func (m *chat) ReadReply(ctx context.Context) (*dto.ChatMessage, error) {
	return m.readWithTimeout(ctx, m.replyTimeout, func(err error) error {
		return &chatErrors.ReplyNotReceivedErr{Err: err}
	})
}

func (m *chat) readWithTimeout(ctx context.Context, timeout time.Duration, wrapTimeout func(error) error) (*dto.ChatMessage, error) {
	readCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		readCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	msg, err := m.ReadOneMessage(readCtx)
	if err == nil {
		return msg, nil
	}

	if wrapTimeout != nil && isReadTimeout(err) && ctx.Err() == nil {
		return nil, wrapTimeout(err)
	}

	return nil, err
}

func (m *chat) SendAck(ctx context.Context, msg *dto.ChatMessage) {
	if msg == nil || msg.GetAck() != nil {
		return
	}

	ack := &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   msg.GetMessageId(),
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: msg.GetProtocolVersion(),
		Payload: &dto.ChatMessage_Ack{
			Ack: &dto.Ack{
				AcknowledgedMessageId: msg.GetMessageId(),
				Status:                dto.AckStatus_ACK_STATUS_ACCEPTED,
				Code:                  dto.ErrorCode_ERROR_CODE_NONE,
			},
		},
	}

	if err := m.WriteOneMessage(ctx, ack); err != nil {
		slog.WarnContext(ctx, "lodging chat",
			"component", "lodging_chat",
			"layer", "transport",
			"event", "ack_send_failed",
			"error", err,
			"message_id", msg.GetMessageId(),
		)
	}
}

func isReadTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || isNetTimeout(err)
}

func isNetTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func outgoingMessageAttrs(msg *dto.ChatMessage) []any {
	attrs := []any{
		"component", "lodging_chat",
		"layer", "transport",
		"event", "message_sent",
		"direction", "outbound",
		"message_id", msg.GetMessageId(),
		"correlation_id", msg.GetCorrelationId(),
		"sender", msg.GetSender().String(),
		"protocol_version", msg.GetProtocolVersion(),
	}

	switch {
	case msg.GetAck() != nil:
		attrs = append(attrs,
			"message_type", "ack",
			"acknowledged_message_id", msg.GetAck().GetAcknowledgedMessageId(),
			"ack_status", msg.GetAck().GetStatus().String(),
			"ack_code", msg.GetAck().GetCode().String(),
		)
	case msg.GetSystemRequest() != dto.SystemRequest_SYSTEM_REQUEST_UNSPECIFIED:
		attrs = append(attrs,
			"message_type", "system_request",
			"system_request", msg.GetSystemRequest().String(),
		)
	case msg.GetSystemNotification() != dto.SystemNotification_SYSTEM_NOTIFICATION_UNSPECIFIED:
		attrs = append(attrs,
			"message_type", "system_notification",
			"system_notification", msg.GetSystemNotification().String(),
		)
	case msg.GetGuestAction() != dto.GuestAction_GUEST_ACTION_UNSPECIFIED:
		attrs = append(attrs,
			"message_type", "guest_action",
			"guest_action", msg.GetGuestAction().String(),
		)
	case msg.GetGuestResponse() != nil:
		attrs = append(attrs,
			"message_type", "guest_response",
			"guest_response_type", guestResponseType(msg.GetGuestResponse()),
		)
	default:
		attrs = append(attrs, "message_type", "unknown")
	}

	return attrs
}

func incomingMessageAttrs(msg *dto.ChatMessage) []any {
	attrs := []any{
		"component", "lodging_chat",
		"layer", "transport",
		"event", "message_received",
		"direction", "inbound",
		"message_id", msg.GetMessageId(),
		"correlation_id", msg.GetCorrelationId(),
		"sender", msg.GetSender().String(),
		"protocol_version", msg.GetProtocolVersion(),
	}

	switch {
	case msg.GetAck() != nil:
		attrs = append(attrs,
			"message_type", "ack",
			"acknowledged_message_id", msg.GetAck().GetAcknowledgedMessageId(),
			"ack_status", msg.GetAck().GetStatus().String(),
			"ack_code", msg.GetAck().GetCode().String(),
		)
	case msg.GetSystemRequest() != dto.SystemRequest_SYSTEM_REQUEST_UNSPECIFIED:
		attrs = append(attrs,
			"message_type", "system_request",
			"system_request", msg.GetSystemRequest().String(),
		)
	case msg.GetSystemNotification() != dto.SystemNotification_SYSTEM_NOTIFICATION_UNSPECIFIED:
		attrs = append(attrs,
			"message_type", "system_notification",
			"system_notification", msg.GetSystemNotification().String(),
		)
	case msg.GetGuestAction() != dto.GuestAction_GUEST_ACTION_UNSPECIFIED:
		attrs = append(attrs,
			"message_type", "guest_action",
			"guest_action", msg.GetGuestAction().String(),
		)
	case msg.GetGuestResponse() != nil:
		attrs = append(attrs,
			"message_type", "guest_response",
			"guest_response_type", guestResponseType(msg.GetGuestResponse()),
		)
	default:
		attrs = append(attrs, "message_type", "unknown")
	}

	return attrs
}

func guestResponseType(response *dto.GuestResponse) string {
	switch response.GetPayload().(type) {
	case *dto.GuestResponse_ShowDocument:
		return "show_document"
	case *dto.GuestResponse_ShowBookingNumber:
		return "show_booking_number"
	case *dto.GuestResponse_ReceiveCottageKey:
		return "receive_cottage_key"
	case *dto.GuestResponse_ReturnCottageKey:
		return "return_cottage_key"
	default:
		return "unknown"
	}
}
