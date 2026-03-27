package helpers

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	clock "github.com/Kenji-Uema/guestManager/internal/infra/clock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MutableClockServer struct {
	clock.UnimplementedClockServiceServer

	mu  sync.RWMutex
	now time.Time
}

func StartClockServer(initialNow time.Time) (host string, port int, server *MutableClockServer, stop func(), err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", 0, nil, nil, err
	}

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return "", 0, nil, nil, errUnexpectedAddrType(ln.Addr())
	}

	server = &MutableClockServer{now: initialNow.UTC()}
	grpcServer := grpc.NewServer()
	clock.RegisterClockServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(ln)
	}()

	stop = func() {
		grpcServer.GracefulStop()
		_ = ln.Close()
	}

	return "127.0.0.1", tcpAddr.Port, server, stop, nil
}

func (s *MutableClockServer) Set(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.now = now.UTC()
}

func (s *MutableClockServer) Now(context.Context, *emptypb.Empty) (*dto.TimeEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &dto.TimeEvent{Time: timestamppb.New(s.now)}, nil
}
