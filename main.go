package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.rtsp-monitor/config"
	"go.rtsp-monitor/metrics"
	"go.rtsp-monitor/prober"
)

func main() {
	configPath := flag.String("config", "./config.yaml", "path to config file")
	secretsPath := flag.String("secrets", "./secrets.yaml", "path to secrets file (optional, for ${VAR} substitution)")
	flag.Parse()

	cfg, err := config.Load(*configPath, *secretsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	initLogging(cfg.Settings.LogLevel)

	slog.Info("loaded config",
		"cameras", len(cfg.Cameras),
		"metric_prefix", cfg.Settings.MetricPrefix,
		"port", cfg.Settings.Port,
	)

	m := metrics.New(cfg.Settings.MetricPrefix)

	srv := metrics.ServeHTTP(cfg.Settings.Port)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	orch := prober.NewOrchestrator(cfg, m)
	orch.Start(ctx)

	<-ctx.Done()
	slog.Info("shutting down...")

	orch.Stop()

	if err := srv.Shutdown(context.Background()); err != nil {
		slog.Error("metrics server shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}

func initLogging(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}
