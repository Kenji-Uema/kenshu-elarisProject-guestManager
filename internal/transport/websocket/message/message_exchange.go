package message

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

var ErrEmptyMessageID = errors.New("chat message must include message_id")

type inflightRequest struct {
	ackCh   chan *dto.ChatMessage
	replyCh chan *dto.ChatMessage
}

type exchange struct {
	conn              *websocket.Conn
	messageRegistry   map[string]*inflightRequest
	messageRegistryMu sync.Mutex
	writeMu           sync.Mutex
}

func NewExchangeIO(conn *websocket.Conn) (*Reader, *Writer) {
	exchange := &exchange{
		conn:            conn,
		messageRegistry: make(map[string]*inflightRequest),
	}

	return &Reader{exchange: exchange}, &Writer{exchange: exchange}
}

func (m *exchange) registerMessage(msg *dto.ChatMessage) (*inflightRequest, string, error) {
	messageID := msg.GetMessageId()
	if messageID == "" {
		return nil, "", ErrEmptyMessageID
	}

	waiter := &inflightRequest{
		ackCh:   make(chan *dto.ChatMessage, 1),
		replyCh: make(chan *dto.ChatMessage, 1),
	}

	m.messageRegistryMu.Lock()
	m.messageRegistry[messageID] = waiter
	m.messageRegistryMu.Unlock()

	return waiter, messageID, nil
}

func (m *exchange) unregisterMessage(messageID string) {
	m.messageRegistryMu.Lock()
	delete(m.messageRegistry, messageID)
	m.messageRegistryMu.Unlock()
}

func (m *exchange) getPendingReply(correlationId string) *inflightRequest {
	m.messageRegistryMu.Lock()
	reply := m.messageRegistry[correlationId]
	m.messageRegistryMu.Unlock()

	return reply
}

func (m *exchange) write(ctx context.Context, msg *dto.ChatMessage) error {
	b, err := protojson.Marshal(msg)
	if err != nil {
		slog.WarnContext(ctx, "websocket ack marshal", "error", err, "message_id", msg.GetMessageId())
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

	return nil
}

func (m *exchange) read(ctx context.Context) (*dto.ChatMessage, error) {
	_, b, err := m.conn.ReadMessage()

	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			slog.WarnContext(ctx, "websocket read", "error", err)
		}
		return nil, err
	}

	var msg dto.ChatMessage
	if err := protojson.Unmarshal(b, &msg); err != nil {
		slog.WarnContext(ctx, "websocket unmarshal", "error", err)
		return nil, err
	}

	return &msg, nil
}
