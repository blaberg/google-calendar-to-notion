package main

import (
	"context"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"google-calendar-to-notion/app"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	var config app.Config
	if err := envconfig.Process("", &config); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	logger, cleanupLogger, err := app.InitLogger(&config)
	if err != nil {
		log.Panic(err)
	}
	defer cleanupLogger()
	logger.Info("initalizing", zap.Any("config", &config))
	app, cleanupApp, err := app.InitApp(ctx, logger, &config)
	if err != nil {
		logger.Panic("failed to initialize", zap.Error(err))
	}
	defer cleanupApp()
	if err := app.Run(ctx); err != nil {
		logger.Error("failed to run", zap.Error(err))
	}
}
