package websocket

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

type Client struct {
	conn    *websocket.Conn
	pending map[string]chan *dto.ChatMessage
	mu      sync.Mutex
	writeMu sync.Mutex
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn:    conn,
		pending: make(map[string]chan *dto.ChatMessage),
	}
}

func (c *Client) SendAndWait(msg *dto.ChatMessage, timeout time.Duration) (*dto.ChatMessage, error) {
	reqID := msg.MessageId
	ch := make(chan *dto.ChatMessage, 1)

	c.mu.Lock()
	c.pending[reqID] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, reqID)
		c.mu.Unlock()
	}()

	b, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}

	c.writeMu.Lock()
	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		c.writeMu.Unlock()
		return nil, err
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		c.writeMu.Unlock()
		return nil, err
	}
	c.writeMu.Unlock()

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// run once in a goroutine
func (c *Client) readLoop(ctx context.Context) {
	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.WarnContext(ctx, "websocket read", "error", err)
			}
			return
		}

		var m dto.ChatMessage
		if err := protojson.Unmarshal(b, &m); err != nil {
			continue
		}

		c.mu.Lock()
		ch := c.pending[m.GetCorrelationId()]
		c.mu.Unlock()

		if ch != nil {
			ch <- &m
		}
	}
}
