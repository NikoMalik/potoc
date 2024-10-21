package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NikoMalik/potoc/internal/app"
	"github.com/NikoMalik/potoc/internal/logger"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	app, err := app.NewApp(ctx)
	if err != nil {
		logger.Fatal("failed to initialize app", zap.Error(err))
	}

	go func() {
		logger.Info("Starting App...",
			zap.String("log_level", app.Config.LogLevel),
			zap.String("host", app.Config.Server.Host),
			zap.String("port", app.Config.Server.Port),
		)
		if err := app.Run(); err != nil {
			logger.Fatal("app failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	timeout := make(chan struct{})

	go func() {
		<-ctx.Done()
		close(timeout)
	}()

	<-quit
	logger.Info("Shutdown Server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Close(shutdownCtx); err != nil {
		logger.Fatal("Server Shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")

}
