package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"stock/internal/scheduler"
	"stock/internal/service/limitdetail"
	"stock/internal/service/monitor"
	"syscall"
	"time"

	"stock/config"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/internal/external/tushare"
	"stock/internal/handler"
	"stock/internal/pkg/logger"
	"stock/internal/ws"
)

func main() {
	if err := config.Init("config/config.yaml"); err != nil {
		log.Fatalf("init config: %v", err)
	}

	cfg := config.GlobalConfig

	if err := initLogger(cfg.Log); err != nil {
		log.Fatalf("init logger: %v", err)
	}

	ctx := context.Background()

	dao.Init()
	cache.Init()

	// Initialize external clients
	tushareClient := tushare.NewClient(&cfg.Tushare, cfg.Retry)

	monitor.Init(tushareClient, &cfg.Monitor)
	limitdetail.Init(tushareClient)
	s := scheduler.NewScheduler(&cfg.Scheduler, monitor.GetInstance())
	s.Setup()
	s.Start()

	hub := ws.NewHub()
	go hub.Run(ctx)

	handlers := handler.NewHandlers(hub, cfg, s)

	// Setup Gin router
	router := setupRouter(cfg, handlers)
	port := resolveServerPort(cfg.Server.Port)

	// Start server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go monitor.GetInstance().Start(ctx)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	s.Stop()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func initLogger(cfg config.LogConfig) error {
	return logger.Init(cfg.Level, cfg.Format, cfg.Output)
}

func resolveServerPort(cfgPort int) string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return fmt.Sprintf("%d", cfgPort)
}
