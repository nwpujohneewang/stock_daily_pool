package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"stock/config"
	"stock/dal/db"
	redis2 "stock/dal/redis"
	"stock/internal/external/tushare"
	"stock/internal/handler"
	"stock/internal/middleware"
	"stock/internal/pkg/logger"
	"stock/internal/service"
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

	db.Init()
	redis2.Init()

	// Initialize external clients
	tushareClient := tushare.NewClient(&cfg.Tushare, cfg.Retry)

	monitorService := service.NewMonitorService(tushareClient, &cfg.Monitor)

	hub := ws.NewHub()
	go hub.Run(ctx)

	handlers := handler.NewHandlers(hub, cfg)

	// Setup Gin router
	router := setupRouter(cfg, handlers)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Printf("server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go monitorService.Start(ctx)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

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

func setupRouter(cfg *config.AppConfig, h *handler.Handlers) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	// Health check - no auth required
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		ws.ServeWS(h.Hub, c.Writer, c.Request)
	})

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(cfg.Server.APIKey))
	{
		pool := api.Group("/pool")
		{
			pool.GET("/limit-up", h.PoolHandler.GetLimitUp)
			pool.GET("/above5", h.PoolHandler.GetAbove5)
			pool.POST("/reclassify", h.PoolHandler.ReclassifyByDate)
		}

		topics := api.Group("/topics")
		{
			topics.GET("", h.TopicHandler.List)
		}

		alerts := api.Group("/alerts")
		{
			alerts.GET("", h.AlertHandler.GetTodayAlerts)
		}

		focus := api.Group("/focus")
		{
			focus.GET("", h.FocusHandler.Get)
			focus.POST("", h.FocusHandler.Set)
			focus.DELETE("/:topic_id", h.FocusHandler.Delete)
		}
	}

	return r
}
