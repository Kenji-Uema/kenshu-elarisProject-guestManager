package mq

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/mqErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
)

func TestRabbitmqConsumerBindQueue(t *testing.T) {
	setupAndRun("consumer bind queue", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqPublisher) {
		t.Run("returns error when queue is not declared", func(t *testing.T) {
			err := consumer.BindQueue(context.Background(), config.BindingConfig{
				ExchangeName: "exchange.does.not.matter",
				RoutingKey:   "rk.test",
			})
			if err == nil {
				t.Fatal("BindQueue() error = nil, want non-nil")
			}
		})

		t.Run("returns error when exchange name is empty", func(t *testing.T) {
			queueName := fmt.Sprintf("consumer.bind.queue.%d", time.Now().UnixNano())
			if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
				t.Fatalf("DeclareQueue() error = %v", err)
			}

			err := consumer.BindQueue(context.Background(), config.BindingConfig{
				ExchangeName: "",
				RoutingKey:   "rk.test",
			})
			if err == nil {
				t.Fatal("BindQueue() error = nil, want non-nil")
			}
		})

		_ = producer
	})
}

func TestRabbitmqProducerPublish(t *testing.T) {
	setupAndRun("producer publish", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqPublisher) {
		exchangeName := fmt.Sprintf("producer.publish.exchange.%d", time.Now().UnixNano())
		if err := producer.DeclareExchange(config.ExchangeConfig{Name: exchangeName, Kind: "direct"}); err != nil {
			t.Fatalf("DeclareExchange() error = %v", err)
		}

		t.Run("returns unexpected error when message cannot be marshaled", func(t *testing.T) {
			invalidMessage := &dto.CleaningRequest{
				RoomName: "invalid-\xff",
				Request:  dto.RequestType_FULL_CLEANING,
			}

			err := producer.Publish(context.Background(), invalidMessage, "cleaning.requested")
			if err == nil {
				t.Fatal("Publish() error = nil, want non-nil")
			}

			var unexpectedErr *mqErrors.UnexpectedErr
			if !errors.As(err, &unexpectedErr) {
				t.Fatalf("Publish() error type = %T, want *mqErrors.UnexpectedErr", err)
			}
		})

		_ = consumer
	})
}

func TestRabbitmqConsumerConsume(t *testing.T) {
	setupAndRun("consumer consume flow", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqPublisher) {
		exchangeName := fmt.Sprintf("consumer.consume.exchange.%d", time.Now().UnixNano())
		queueName := fmt.Sprintf("consumer.consume.queue.%d", time.Now().UnixNano())
		routingKey := "cleaning.requested"

		if err := producer.DeclareExchange(config.ExchangeConfig{Name: exchangeName, Kind: "direct"}); err != nil {
			t.Fatalf("DeclareExchange() error = %v", err)
		}
		if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
			t.Fatalf("DeclareQueue() error = %v", err)
		}
		if err := consumer.BindQueue(context.Background(), config.BindingConfig{ExchangeName: exchangeName, RoutingKey: routingKey}); err != nil {
			t.Fatalf("BindQueue() error = %v", err)
		}

		consumeCtx, cancelConsume := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelConsume()

		deliveries, err := consumer.Consume(consumeCtx)
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}

		msg := &dto.CleaningRequest{
			RoomName: "cottage-7",
			Request:  dto.RequestType_PREPARE_FOR_GUEST,
		}

		if err := producer.Publish(context.Background(), msg, routingKey); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		select {
		case d, ok := <-deliveries:
			if !ok {
				t.Fatal("deliveries channel closed before receiving message")
			}
			if d.RoutingKey != routingKey {
				t.Fatalf("delivery routing key = %q, want %q", d.RoutingKey, routingKey)
			}
			if string(d.Body) == "" {
				t.Fatal("delivery body is empty")
			}
			if err := d.Ack(false); err != nil {
				t.Fatalf("Ack() error = %v", err)
			}
		case <-consumeCtx.Done():
			t.Fatalf("timed out waiting for delivery: %v", consumeCtx.Err())
		}

		cancelConsume()
		select {
		case _, ok := <-deliveries:
			if ok {
				t.Fatal("expected deliveries channel to close after cancel")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for deliveries channel to close")
		}
	})
}
