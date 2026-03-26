package clock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewClockEmu(t *testing.T) {
	clockClient, err := NewClockEmu(config.Services{
		ClockSimulator: struct {
			GrpcHost string `env:"CLOCK_EMU_GRPC_HOST,required"`
			GrpcPort int    `env:"CLOCK_EMU_GRPC_PORT,required"`
		}{
			GrpcHost: "localhost",
			GrpcPort: 50051,
		},
	})
	if err != nil {
		t.Fatalf("NewClockEmu() error = %v", err)
	}
	if clockClient == nil {
		t.Fatal("NewClockEmu() returned nil client")
	}

	closer, ok := clockClient.(interface{ Close() error })
	if !ok {
		t.Fatalf("NewClockEmu() returned %T, want client with Close()", clockClient)
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClientNow(t *testing.T) {
	t.Run("returns mapped time from grpc response", func(t *testing.T) {
		want := time.Date(2026, 3, 24, 15, 4, 5, 0, time.UTC)
		rpcClient := &stubClockServiceClient{
			nowFn: func(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*dto.TimeEvent, error) {
				return &dto.TimeEvent{Time: timestamppb.New(want)}, nil
			},
		}

		clockClient := &client{client: rpcClient}
		got, err := clockClient.Now(context.Background())
		if err != nil {
			t.Fatalf("Now() error = %v", err)
		}
		if got == nil {
			t.Fatal("Now() returned nil time")
		}
		if !got.Equal(want) {
			t.Fatalf("Now() = %v, want %v", *got, want)
		}
		if rpcClient.nowCallCount != 1 {
			t.Fatalf("ClockServiceClient.Now() call count = %d, want 1", rpcClient.nowCallCount)
		}
	})

	t.Run("returns grpc error", func(t *testing.T) {
		wantErr := errors.New("clock rpc failed")
		rpcClient := &stubClockServiceClient{
			nowFn: func(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*dto.TimeEvent, error) {
				return nil, wantErr
			},
		}

		clockClient := &client{client: rpcClient}
		got, err := clockClient.Now(context.Background())
		if got != nil {
			t.Fatalf("Now() time = %v, want nil", got)
		}
		if !errors.Is(err, wantErr) {
			t.Fatalf("Now() error = %v, want %v", err, wantErr)
		}
	})
}

func TestClientClose(t *testing.T) {
	conn, err := grpc.NewClient(
		"passthrough:///clock-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	clockClient := &client{conn: conn}
	if err := clockClient.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

type stubClockServiceClient struct {
	nowFn        func(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*dto.TimeEvent, error)
	nowCallCount int
}

func (s *stubClockServiceClient) Now(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*dto.TimeEvent, error) {
	s.nowCallCount++
	if s.nowFn != nil {
		return s.nowFn(ctx, in, opts...)
	}
	return nil, nil
}

func (s *stubClockServiceClient) Header() (metadata.MD, error) { return nil, nil }
