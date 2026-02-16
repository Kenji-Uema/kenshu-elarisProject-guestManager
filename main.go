package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kenji-Uema/guestManager/internal/bootstrap"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/tooling/log"
)

func exitOnError(ctx context.Context, errMsg string, err error) {
	if err != nil {
		slog.ErrorContext(ctx, errMsg, "error", err)
		os.Exit(1)
	}
}

func main() {
	slog.SetDefault(log.NewLogger())
	baseCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	configs, err := config.LoadConfigs()
	exitOnError(baseCtx, "failed to load configs", err)

	application, err := bootstrap.BuildApp(baseCtx, configs)
	exitOnError(baseCtx, "failed to build app", err)
	defer func() {
		if err := application.Close(baseCtx); err != nil {
			slog.ErrorContext(baseCtx, "shutdown app", "error", err)
		}
	}()

	slog.InfoContext(baseCtx, "starting guestManager")
	err = application.Router.Run(fmt.Sprintf("%s:%d", configs.AppConfig.Host, configs.AppConfig.Port))
	exitOnError(baseCtx, "failed to run http server", err)
}
