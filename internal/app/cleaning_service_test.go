package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	mqfakes "github.com/Kenji-Uema/guestManager/internal/infra/mq/fakes"
	"google.golang.org/protobuf/proto"
)

func TestNewCleaningService(t *testing.T) {
	t.Run("returns error when publisher is nil", func(t *testing.T) {
		var publisher *mqfakes.FakeMqProducer

		service, err := NewCleaningService(publisher)
		if err == nil {
			t.Fatal("NewCleaningService() error = nil, want non-nil")
		}
		if service != nil {
			t.Fatalf("NewCleaningService() service = %v, want nil", service)
		}
	})

	t.Run("returns service when publisher is valid", func(t *testing.T) {
		publisher := &mqfakes.FakeMqProducer{}

		service, err := NewCleaningService(publisher)
		if err != nil {
			t.Fatalf("NewCleaningService() error = %v", err)
		}
		if service == nil {
			t.Fatal("NewCleaningService() service = nil, want non-nil")
		}
	})
}

func TestCleaningServiceCleanRoom(t *testing.T) {
	tests := []struct {
		name        string
		requestType enum.CleaningRequestType
		wantType    dto.RequestType
	}{
		{
			name:        "publishes prepare for guest request",
			requestType: enum.PrepareForGuest,
			wantType:    dto.RequestType_PREPARE_FOR_GUEST,
		},
		{
			name:        "publishes daily cleaning request",
			requestType: enum.DailyCleaning,
			wantType:    dto.RequestType_DAILY_CLEANING,
		},
		{
			name:        "publishes full cleaning request",
			requestType: enum.FullCleaning,
			wantType:    dto.RequestType_FULL_CLEANING,
		},
		{
			name:        "publishes prepare for sleep request",
			requestType: enum.PrepareForSleep,
			wantType:    dto.RequestType_PREPARE_FOR_SLEEP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &mqfakes.FakeMqProducer{}
			service, err := NewCleaningService(publisher)
			if err != nil {
				t.Fatalf("NewCleaningService() error = %v", err)
			}

			req, err := domain.NewCleaningOrder("Cottage 1", tt.requestType)
			if err != nil {
				t.Fatalf("NewCleaningOrder() error = %v", err)
			}

			ctx := context.Background()
			if err := service.CleanRoom(ctx, req); err != nil {
				t.Fatalf("CleanRoom() error = %v", err)
			}

			if publisher.PublishCallCount != 1 {
				t.Fatalf("Publish() call count = %d, want 1", publisher.PublishCallCount)
			}
			if publisher.LastPublishedCtx != ctx {
				t.Fatal("Publish() ctx did not match input ctx")
			}
			if publisher.LastPublishedRoutingKey != "" {
				t.Fatalf("Publish() routing key = %q, want empty string", publisher.LastPublishedRoutingKey)
			}

			message, ok := publisher.LastPublishedMessage.(*dto.CleaningRequest)
			if !ok {
				t.Fatalf("Publish() message type = %T, want *dto.CleaningOrder", publisher.LastPublishedMessage)
			}
			if message.RoomName != "Cottage 1" {
				t.Fatalf("Publish() room name = %q, want %q", message.RoomName, "Cottage 1")
			}
			if message.Request != tt.wantType {
				t.Fatalf("Publish() request type = %v, want %v", message.Request, tt.wantType)
			}
		})
	}

	t.Run("returns publish error", func(t *testing.T) {
		publishErr := errors.New("publish failed")
		publisher := &mqfakes.FakeMqProducer{
			PublishFn: func(ctx context.Context, message proto.Message, routingKey string) error {
				return publishErr
			},
		}
		service, err := NewCleaningService(publisher)
		if err != nil {
			t.Fatalf("NewCleaningService() error = %v", err)
		}

		req, err := domain.NewCleaningOrder("Cottage 2", enum.DailyCleaning)
		if err != nil {
			t.Fatalf("NewCleaningOrder() error = %v", err)
		}

		err = service.CleanRoom(context.Background(), req)
		if !errors.Is(err, publishErr) {
			t.Fatalf("CleanRoom() error = %v, want %v", err, publishErr)
		}
	})
}
