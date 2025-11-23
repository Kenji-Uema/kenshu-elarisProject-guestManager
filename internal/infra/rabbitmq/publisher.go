package rabbitmq

import (
	"context"
	"fmt"
	"guestManager/internal/config"
	"guestManager/internal/domain"
	"guestManager/internal/port"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type publisher struct {
	channel *amqp.Channel
}

func NewPublisher(client *client, cfg config.CleaningExchangeConfig) (port.MqPublisher, error) {
	if err := client.initPublisher(cfg); err != nil {
		return nil, err
	}

	return &publisher{channel: client.channel}, nil
}

func (p *publisher) Publish(ctx context.Context, message domain.RabbitMqMessage) error {
	confirmations := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err := p.channel.PublishWithContext(ctx,
		message.Exchange,
		message.Key,
		false, false,
		amqp.Publishing{
			ContentType:  message.ContentType,
			Body:         message.Body,
			DeliveryMode: amqp.Persistent,
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
