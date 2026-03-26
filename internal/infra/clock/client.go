package clock

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type client struct {
	conn   *grpc.ClientConn
	client ClockServiceClient
}

func NewClockEmu(cfg config.Services) (port.ClockClient, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.ClockSimulator.GrpcHost, cfg.ClockSimulator.GrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &client{conn: conn, client: NewClockServiceClient(conn)}, nil
}

func (e *client) Close() error {
	return e.conn.Close()
}

func (e *client) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := e.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}
