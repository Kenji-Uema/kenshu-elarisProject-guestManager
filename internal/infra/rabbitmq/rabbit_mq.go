package rabbitmq

import (
	"guestManager/internal/config"
	"guestManager/internal/port"

	amqp "github.com/rabbitmq/amqp091-go"
)

type client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMqClient(config config.RabbitMqConfig) (port.MqClient, error) {
	conn, err := amqp.Dial(config.Url)
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

	if err := ch.Confirm(config.ConfirmNotWait); err != nil {
		return nil, err
	}

	return &client{conn: conn, channel: ch}, nil
}

func (c *client) CloseChannel() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *client) EnsurePublisherTopology(config config.CleaningExchangeConfig) error {
	return c.channel.ExchangeDeclare(
		config.Name, config.Kind, config.Durable, config.AutoDelete, config.Internal, config.NoWait, nil,
	)
}

func (c *client) initPublisher(exchangeConfig config.CleaningExchangeConfig) error {
	if err := c.EnsurePublisherTopology(exchangeConfig); err != nil {
		return err
	}

	return nil
}
