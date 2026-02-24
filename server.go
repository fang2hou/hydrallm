package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the proxy server",
		Run:   runServe,
	}
}

func runServe(_ *cobra.Command, _ []string) {
	cfg, err := loadConfig()
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	logger.Info(
		"starting hydrallm",
		"host",
		cfg.Server.Host,
		"port",
		cfg.Server.Port,
		"models",
		len(cfg.Models),
	)
	for i, m := range cfg.Models {
		logger.Info(
			"configured model",
			"index",
			i,
			"endpoint",
			m.Endpoint,
			"model",
			m.Model,
			"type",
			m.Type,
			"attempts",
			m.Attempts,
		)
	}

	proxy := newProxy(cfg, logger)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           proxy,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("failed to start server: %v", err)
		}
	}()

	logger.Info("hydrallm listening", "address", server.Addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Error("server shutdown error", "error", err)
	}
}
