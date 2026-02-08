package mq

import (
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMqConnection struct {
	conn *amqp.Connection
}

func NewRabbitMqConnection(cfg config.RabbitMqConfig) (*RabbitMqConnection, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.Username, cfg.Password, cfg.Host, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	return &RabbitMqConnection{conn: conn}, nil
}

func (c *RabbitMqConnection) IsConnectionOpen() bool {
	return !c.conn.IsClosed()
}

func (c *RabbitMqConnection) Close() error {
	return c.conn.Close()
}
