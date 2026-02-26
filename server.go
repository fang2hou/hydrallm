package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
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

	logger.Info("starting hydrallm", "listeners", len(cfg.Listeners))

	// Create servers for each listener
	servers := make([]*http.Server, 0, len(cfg.Listeners))
	for i := range cfg.Listeners {
		l := &cfg.Listeners[i]

		logger.Info(
			"configured listener",
			"name",
			l.Name,
			"host",
			l.Host,
			"port",
			l.Port,
			"models",
			len(l.Models),
		)
		for _, m := range l.ResolvedModels {
			logger.Info(
				"configured model",
				"listener",
				l.Name,
				"provider",
				m.Provider,
				"model",
				m.Model,
				"type",
				m.Type,
				"attempts",
				m.Attempts,
			)
		}

		proxy := newProxy(l, cfg, logger)

		server := &http.Server{
			Addr:              fmt.Sprintf("%s:%d", l.Host, l.Port),
			Handler:           proxy,
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       l.ReadTimeout,
			WriteTimeout:      l.WriteTimeout,
		}
		servers = append(servers, server)
	}

	// Start all servers
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalf("failed to start server %s: %v", s.Addr, err)
			}
		}(server)
		logger.Info("hydrallm listening", "address", server.Addr)
	}

	// Wait for shutdown signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("shutting down servers...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownWg sync.WaitGroup
	for _, server := range servers {
		shutdownWg.Add(1)
		go func(s *http.Server) {
			defer shutdownWg.Done()
			if err := s.Shutdown(shutdownCtx); err != nil {
				logger.Error("server shutdown error", "address", s.Addr, "error", err)
			}
		}(server)
	}
	shutdownWg.Wait()

	wg.Wait()
	logger.Info("all servers stopped")
}
