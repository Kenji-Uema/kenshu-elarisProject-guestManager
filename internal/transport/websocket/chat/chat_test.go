package chat

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/chatErrors"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestChatWriteOneMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns error when message id is empty", func(t *testing.T) {
		m := &chat{conn: &fakeConn{}}

		err := m.WriteOneMessage(context.Background(), &dto.ChatMessage{})
		if !errors.Is(err, ErrEmptyMessageID) {
			t.Fatalf("WriteOneMessage() expected ErrEmptyMessageID, got %v", err)
		}
	})

	t.Run("writes marshaled protobuf as websocket text message", func(t *testing.T) {
		conn := &fakeConn{}
		m := &chat{conn: conn}
		msg := &dto.ChatMessage{
			MessageId:       "message-1",
			CorrelationId:   "corr-1",
			ProtocolVersion: "lodging.v1",
			Sender:          dto.Sender_SENDER_SYSTEM,
			Payload: &dto.ChatMessage_SystemNotification{
				SystemNotification: dto.SystemNotification_BREAKFAST_READY,
			},
		}

		if err := m.WriteOneMessage(context.Background(), msg); err != nil {
			t.Fatalf("WriteOneMessage() unexpected error: %v", err)
		}

		if conn.writeMessageType != websocket.TextMessage {
			t.Fatalf("WriteOneMessage() expected websocket text message, got %d", conn.writeMessageType)
		}
		if conn.writeDeadline.IsZero() {
			t.Fatal("WriteOneMessage() expected write deadline to be set")
		}

		var got dto.ChatMessage
		if err := protojson.Unmarshal(conn.writePayload, &got); err != nil {
			t.Fatalf("WriteOneMessage() payload should be valid json: %v", err)
		}
		if got.GetMessageId() != msg.GetMessageId() {
			t.Fatalf("WriteOneMessage() unexpected message id: %q", got.GetMessageId())
		}
		if got.GetSystemNotification() != dto.SystemNotification_BREAKFAST_READY {
			t.Fatalf("WriteOneMessage() unexpected notification payload: %v", got.GetSystemNotification())
		}
	})

	t.Run("returns connection write errors", func(t *testing.T) {
		wantErr := errors.New("write failed")
		conn := &fakeConn{writeErr: wantErr}
		m := &chat{conn: conn}

		err := m.WriteOneMessage(context.Background(), &dto.ChatMessage{
			MessageId: "message-1",
			Payload: &dto.ChatMessage_Ack{
				Ack: &dto.Ack{},
			},
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("WriteOneMessage() expected write error, got %v", err)
		}
	})
}

func TestChatReadOneMessage(t *testing.T) {
	t.Parallel()

	t.Run("uses context deadline and unmarshals message", func(t *testing.T) {
		payload, err := protojson.Marshal(&dto.ChatMessage{
			MessageId:     "guest-message-1",
			CorrelationId: "corr-1",
			Payload: &dto.ChatMessage_GuestAction{
				GuestAction: dto.GuestAction_SHOW_FOR_CHECKIN,
			},
		})
		if err != nil {
			t.Fatalf("protojson.Marshal() unexpected error: %v", err)
		}

		conn := &fakeConn{readPayload: payload}
		m := &chat{conn: conn}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
		defer cancel()

		msg, readErr := m.ReadOneMessage(ctx)
		if readErr != nil {
			t.Fatalf("ReadOneMessage() unexpected error: %v", readErr)
		}
		if msg.GetMessageId() != "guest-message-1" {
			t.Fatalf("ReadOneMessage() unexpected message id: %q", msg.GetMessageId())
		}
		if msg.GetGuestAction() != dto.GuestAction_SHOW_FOR_CHECKIN {
			t.Fatalf("ReadOneMessage() unexpected guest action: %v", msg.GetGuestAction())
		}

		deadline, _ := ctx.Deadline()
		if !conn.readDeadline.Equal(deadline) {
			t.Fatalf("ReadOneMessage() expected read deadline %v, got %v", deadline, conn.readDeadline)
		}
	})

	t.Run("clears read deadline when context has no deadline", func(t *testing.T) {
		payload, err := protojson.Marshal(&dto.ChatMessage{MessageId: "guest-message-1"})
		if err != nil {
			t.Fatalf("protojson.Marshal() unexpected error: %v", err)
		}

		conn := &fakeConn{readPayload: payload}
		m := &chat{conn: conn}

		if _, err := m.ReadOneMessage(context.Background()); err != nil {
			t.Fatalf("ReadOneMessage() unexpected error: %v", err)
		}
		if !conn.readDeadline.IsZero() {
			t.Fatalf("ReadOneMessage() expected zero read deadline, got %v", conn.readDeadline)
		}
	})

	t.Run("returns unmarshal errors", func(t *testing.T) {
		m := &chat{conn: &fakeConn{readPayload: []byte("{not json")}}

		_, err := m.ReadOneMessage(context.Background())
		if err == nil {
			t.Fatal("ReadOneMessage() expected unmarshal error, got nil")
		}
	})

	t.Run("returns connection read errors", func(t *testing.T) {
		wantErr := errors.New("read failed")
		m := &chat{conn: &fakeConn{readErr: wantErr}}

		_, err := m.ReadOneMessage(context.Background())
		if !errors.Is(err, wantErr) {
			t.Fatalf("ReadOneMessage() expected read error, got %v", err)
		}
	})
}

func TestChatReadAckAndReply(t *testing.T) {
	t.Parallel()

	t.Run("wraps ack read timeout", func(t *testing.T) {
		m := &chat{
			conn:       &fakeConn{readErr: timeoutErr{}},
			ackTimeout: 10 * time.Millisecond,
		}

		_, err := m.ReadAck(context.Background())
		var ackErr *chatErrors.AckNotReceivedErr
		if !errors.As(err, &ackErr) {
			t.Fatalf("ReadAck() expected AckNotReceivedErr, got %v", err)
		}
	})

	t.Run("wraps reply read timeout", func(t *testing.T) {
		m := &chat{
			conn:         &fakeConn{readErr: timeoutErr{}},
			replyTimeout: 10 * time.Millisecond,
		}

		_, err := m.ReadReply(context.Background())
		var replyErr *chatErrors.ReplyNotReceivedErr
		if !errors.As(err, &replyErr) {
			t.Fatalf("ReadReply() expected ReplyNotReceivedErr, got %v", err)
		}
	})

	t.Run("does not wrap timeout when parent context is already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		m := &chat{
			conn:       &fakeConn{readErr: context.DeadlineExceeded},
			ackTimeout: 10 * time.Millisecond,
		}

		_, err := m.ReadAck(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ReadAck() expected context deadline exceeded, got %v", err)
		}

		var ackErr *chatErrors.AckNotReceivedErr
		if errors.As(err, &ackErr) {
			t.Fatalf("ReadAck() expected raw context error, got wrapped error %v", err)
		}
	})
}

func TestChatSendAck(t *testing.T) {
	t.Parallel()

	t.Run("writes ack for non ack message", func(t *testing.T) {
		conn := &fakeConn{}
		m := &chat{conn: conn}
		msg := &dto.ChatMessage{
			MessageId:       "guest-message-1",
			ProtocolVersion: "lodging.v1",
			Payload: &dto.ChatMessage_GuestAction{
				GuestAction: dto.GuestAction_SHOW_FOR_CHECKIN,
			},
		}

		m.SendAck(context.Background(), msg)

		var ack dto.ChatMessage
		if err := protojson.Unmarshal(conn.writePayload, &ack); err != nil {
			t.Fatalf("SendAck() payload should be valid json: %v", err)
		}
		if ack.GetAck() == nil {
			t.Fatalf("SendAck() expected ack payload, got %+v", &ack)
		}
		if ack.GetCorrelationId() != msg.GetMessageId() {
			t.Fatalf("SendAck() expected correlation id %q, got %q", msg.GetMessageId(), ack.GetCorrelationId())
		}
		if ack.GetAck().GetAcknowledgedMessageId() != msg.GetMessageId() {
			t.Fatalf("SendAck() expected acknowledged message id %q, got %q", msg.GetMessageId(), ack.GetAck().GetAcknowledgedMessageId())
		}
		if ack.GetSender() != dto.Sender_SENDER_SYSTEM {
			t.Fatalf("SendAck() expected system sender, got %v", ack.GetSender())
		}
		if ack.GetProtocolVersion() != msg.GetProtocolVersion() {
			t.Fatalf("SendAck() expected protocol version %q, got %q", msg.GetProtocolVersion(), ack.GetProtocolVersion())
		}
		if ack.GetAck().GetStatus() != dto.AckStatus_ACK_STATUS_ACCEPTED {
			t.Fatalf("SendAck() expected accepted status, got %v", ack.GetAck().GetStatus())
		}
	})

	t.Run("ignores nil and ack messages", func(t *testing.T) {
		conn := &fakeConn{}
		m := &chat{conn: conn}

		m.SendAck(context.Background(), nil)
		m.SendAck(context.Background(), &dto.ChatMessage{
			MessageId: "ack-1",
			Payload: &dto.ChatMessage_Ack{
				Ack: &dto.Ack{AcknowledgedMessageId: "message-1"},
			},
		})

		if len(conn.writePayload) != 0 {
			t.Fatalf("SendAck() expected no writes, got %q", string(conn.writePayload))
		}
	})
}

func TestIsReadTimeout(t *testing.T) {
	t.Parallel()

	if !isReadTimeout(context.DeadlineExceeded) {
		t.Fatal("isReadTimeout() expected context deadline exceeded to be treated as timeout")
	}
	if !isReadTimeout(timeoutErr{}) {
		t.Fatal("isReadTimeout() expected net timeout to be treated as timeout")
	}
	if isReadTimeout(errors.New("boom")) {
		t.Fatal("isReadTimeout() did not expect generic error to be treated as timeout")
	}
}

type fakeConn struct {
	writeDeadline    time.Time
	writePayload     []byte
	writeMessageType int
	writeErr         error
	setWriteErr      error

	readDeadline time.Time
	readPayload  []byte
	readErr      error
	setReadErr   error
}

func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	f.writeDeadline = t
	return f.setWriteErr
}

func (f *fakeConn) WriteMessage(messageType int, data []byte) error {
	f.writeMessageType = messageType
	f.writePayload = append([]byte(nil), data...)
	return f.writeErr
}

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	f.readDeadline = t
	return f.setReadErr
}

func (f *fakeConn) ReadMessage() (messageType int, p []byte, err error) {
	if f.readErr != nil {
		return 0, nil, f.readErr
	}
	return websocket.TextMessage, append([]byte(nil), f.readPayload...), nil
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

var _ net.Error = timeoutErr{}
