//package probe
//
//import (
//	"context"
//	"fmt"
//	"log/slog"
//	"net/http"
//	"time"
//
//	"github.com/Kenji-Uema/clockEmulator/internal/config"
//	"github.com/Kenji-Uema/clockEmulator/internal/transport/grpc/clock/pb/clockEmu"
//	clockEmuProto "github.com/Kenji-Uema/guestManager/internal/transport/grpc/clock/pb/clockEmu"
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/credentials/insecure"
//	"google.golang.org/protobuf/types/known/emptypb"
//)
//
//func HealthHandler(cfg config.GrpcConfig) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
//		defer cancel()
//
//		if err := checkGrpcNow(ctx, cfg); err != nil {
//			http.Error(w, "grpc now failed", http.StatusServiceUnavailable)
//			return
//		}
//		w.WriteHeader(http.StatusOK)
//	}
//}
//
//func checkGrpcNow(ctx context.Context, cfg config.GrpcConfig) error {
//	conn, err := grpc.NewClient(
//		fmt.Sprintf("%s:%d", cfg.Address, cfg.Port),
//		grpc.WithTransportCredentials(insecure.NewCredentials()),
//	)
//	if err != nil {
//		return err
//	}
//	defer func(conn *grpc.ClientConn) {
//		err := conn.Close()
//		if err != nil {
//			slog.Error("close grpc connection", "error", err)
//		}
//	}(conn)
//
//	client := clockEmuProto.NewClockServiceClient(conn)
//	_, err = client.Now(ctx, &emptypb.Empty{})
//	return err
//}
