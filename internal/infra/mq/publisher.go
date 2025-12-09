package mq

import (
	"context"
	"fmt"
	"guestManager/internal/config"
	"guestManager/internal/domain"
	"guestManager/internal/port"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMqPublisher struct {
	channel       *amqp.Channel
	confirmations <-chan amqp.Confirmation
}

func NewRabbitMqPublisher(client *RabbitMqConnection, cfg config.CleaningExchangeConfig) (port.MqPublisher, error) {
	channel, err := client.conn.Channel()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("create channel: %w", err)
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("enable confirm mode: %w", err)
	}

	if err = channel.ExchangeDeclare(cfg.Name, cfg.Kind, cfg.Durable, cfg.AutoDelete, cfg.Internal, cfg.NoWait, nil); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	confirmations := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	return &rabbitMqPublisher{channel: channel, confirmations: confirmations}, nil
}

func (p *rabbitMqPublisher) Publish(ctx context.Context, message domain.RabbitMqMessage) error {
	err := p.channel.PublishWithContext(ctx,
		message.Exchange(),
		message.Key(),
		false, false,
		amqp.Publishing{
			ContentType:  message.ContentType(),
			Body:         message.Body(),
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return err
	}

	select {
	case conf := <-p.confirmations:
		if !conf.Ack {
			return fmt.Errorf("publish not acknowledged")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *rabbitMqPublisher) Close() error {
	return p.channel.Close()
}
