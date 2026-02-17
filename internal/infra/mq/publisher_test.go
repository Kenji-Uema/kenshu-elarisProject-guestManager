package mq

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

var (
	rabbitmqContainer *rabbitmq.RabbitMQContainer
	mqConnection      *RabbitMqConnection
	exchangeCfg       = config.CleaningExchangeConfig{
		Name:       "ex.cleaning",
		Kind:       "direct",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	rabbitmqContainer, err = rabbitmq.Run(ctx,
		"rabbitmq:3.12.11-management-alpine",
		rabbitmq.WithAdminUsername("admin"),
		rabbitmq.WithAdminPassword("password"),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	amqpURL, err := rabbitmqContainer.AmqpURL(ctx)
	if err != nil {
		log.Fatalf("failed to get AMQP URL: %s", err)
	}

	parsedURL, err := url.Parse(amqpURL)
	if err != nil {
		log.Fatalf("failed to parse AMQP URL: %s", err)
	}
	password, _ := parsedURL.User.Password()

	mqConnection, err = NewRabbitMqConnection(config.RabbitMqConfig{
		Username: config.Secret(parsedURL.User.Username()),
		Password: config.Secret(password),
		Host:     parsedURL.Hostname(),
		Port: func() int {
			if parsedURL.Port() == "" {
				return 5672
			}
			p, convErr := strconv.Atoi(parsedURL.Port())
			if convErr != nil {
				log.Fatalf("failed to parse AMQP port: %s", convErr)
			}
			return p
		}(),
	})
	if err != nil {
		log.Fatalf("failed to create rabbitmq connection: %s", err)
	}

	code := m.Run()

	if err := mqConnection.Close(); err != nil {
		log.Fatalf("failed to close rabbitmq mqConnection: %s", err)
	}

	if err := testcontainers.TerminateContainer(rabbitmqContainer); err != nil {
		log.Fatalf("failed to terminate container: %s", err)
	}

	os.Exit(code)
}

func newPublisher(t *testing.T) *rabbitMqPublisher {
	t.Helper()

	pub, err := NewRabbitMqPublisher(mqConnection, exchangeCfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	rmqPub, ok := pub.(*rabbitMqPublisher)
	if !ok {
		t.Fatalf("unexpected publisher type %T", pub)
	}

	t.Cleanup(func() {
		_ = rmqPub.Close()
	})

	return rmqPub
}

func TestPublisher_PublishSendsMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	publisher := newPublisher(t)

	queue, err := publisher.channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("failed to declare queue: %v", err)
	}

	if err := publisher.channel.QueueBind(queue.Name, "room.ready", exchangeCfg.Name, false, nil); err != nil {
		t.Fatalf("failed to bind queue: %v", err)
	}

	t.Cleanup(func() {
		if _, err := publisher.channel.QueueDelete(queue.Name, false, false, false); err != nil {
			t.Logf("cleanup: failed to delete queue %s: %v", queue.Name, err)
		}
		if err := publisher.channel.ExchangeDelete(exchangeCfg.Name, false, false); err != nil {
			t.Logf("cleanup: failed to delete exchange %s: %v", exchangeCfg.Name, err)
		}
	})

	message, err := domain.NewRabbitMqMessage(exchangeCfg.Name, "room.ready", "application/json", []byte("hello"))
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	if err := publisher.Publish(ctx, message); err != nil {
		t.Fatalf("Publish() returned error: %v", err)
	}

	deliveries, err := publisher.channel.Consume(queue.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatalf("failed to start consumer: %v", err)
	}

	select {
	case delivery := <-deliveries:
		if string(delivery.Body) != "hello" {
			t.Fatalf("expected body 'hello', got %s", string(delivery.Body))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for published message")
	}
}

func TestPublisher_PublishContextCanceled(t *testing.T) {
	publisher := newPublisher(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	message, err := domain.NewRabbitMqMessage(exchangeCfg.Name, "room.ready", "application/json", []byte("body"))
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	if err := publisher.Publish(ctx, message); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestPublisher_PublishOnClosedChannelFails(t *testing.T) {
	publisher := newPublisher(t)
	_ = publisher.Close()

	message, err := domain.NewRabbitMqMessage(exchangeCfg.Name, "room.ready", "application/json", []byte("body"))
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	if err := publisher.Publish(context.Background(), message); err == nil {
		t.Fatal("expected error when publishing on closed channel, got nil")
	}
}
