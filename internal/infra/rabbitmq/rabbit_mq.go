package rabbitmq

import (
	"context"
	"fmt"
	"guestManager/internal/port"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func New(url string) (port.MqClient, error) {
	conn, err := amqp.Dial(url) // e.g. "amqp://guest:guest@localhost:5672/"
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		err := conn.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	return &Client{conn: conn, channel: ch}, nil
}

func (c *Client) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Client) EnsureTopology() error {
	// durable direct exchange
	if err := c.channel.ExchangeDeclare(
		"guest.events", "direct", true, false, false, false, nil,
	); err != nil {
		return err
	}

	// durable queue
	q, err := c.channel.QueueDeclare(
		"guest.created.q", true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	// bind queue to exchange with routing key
	return c.channel.QueueBind(q.Name, "guest.created", "guest.events", false, nil)
}

func (c *Client) EnablePublisherConfirms() error {
	return c.channel.Confirm(false) // turn on confirms for the whole channel
}

func (c *Client) Publish(ctx context.Context, body []byte) error {
	confirmations := c.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err := c.channel.PublishWithContext(ctx,
		"guest.events",  // exchange
		"guest.created", // routing key
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // make messages durable
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return err
	}

	select {
	case conf := <-confirmations:
		if !conf.Ack {
			return fmt.Errorf("publish not acknowledged")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
