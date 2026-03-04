package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Loszect1/Ecommerce---BE-Golang/app"
	"github.com/Loszect1/Ecommerce---BE-Golang/config"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
)

func main() {
	cfg := config.FromEnv()
	log := logger.New()

	handler := app.New(cfg)

	addr := ":" + cfg.HTTPPort
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("starting http server", map[string]any{"addr": addr, "env": cfg.Env})

	// Graceful shutdown.
	idleConnsClosed := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info("received shutdown signal", map[string]any{"signal": sig.String()})

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("http server shutdown error", err, nil)
		}
		close(idleConnsClosed)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("http server error", err, map[string]any{"addr": addr})
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
	}

	<-idleConnsClosed
	log.Info("server stopped", nil)
}

