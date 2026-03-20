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
	"github.com/redis/go-redis/v9"
	"stock/dal/db"
	"stock/internal/cache"
	"stock/internal/config"
	"stock/internal/external/llm"
	"stock/internal/external/tushare"
	"stock/internal/handler"
	"stock/internal/middleware"
	"stock/internal/pkg/logger"
	"stock/internal/repo"
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

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	// Initialize external clients
	tushareClient := tushare.NewClient(&cfg.Tushare, cfg.Retry)
	llmClient := llm.NewClient(&cfg.LLM)

	quoteCache := cache.NewQuoteCache(redisClient)
	poolCache := cache.NewPoolCache(redisClient)
	focusCache := cache.NewFocusCache(redisClient)
	mappingCache := cache.NewMappingCache(redisClient)
	conceptCache := cache.NewConceptCache(redisClient)

	stockRepo := repo.NewStockRepo(db.DB)
	boardRepo := repo.NewBoardRepo(db.DB)
	alertRepo := repo.NewAlertRepo(db.DB)
	poolRepo := repo.NewPoolRepo(db.DB)
	mappingRepo := repo.NewMappingRepo(db.DB)
	topicRepo := repo.NewTopicRepo(db.DB)
	evidenceRepo := repo.NewEvidenceRepo(db.DB)

	quoteFetcher := service.NewQuoteFetcher(tushareClient, quoteCache, stockRepo, cfg.Monitor.ShardCount)

	classifySvc := service.NewClassifyService(
		mappingCache,
		conceptCache,
		topicRepo,
		mappingRepo,
		evidenceRepo,
	)

	alertSvc := service.NewAlertService(alertRepo, focusCache, poolRepo, &cfg.Monitor)

	monitorService := service.NewMonitorService(
		stockRepo,
		quoteCache,
		poolCache,
		boardRepo,
		poolRepo,
		quoteFetcher,
		classifySvc,
		alertSvc,
		&cfg.Monitor,
	)

	hub := ws.NewHub()
	go hub.Run(ctx)

	handlers := handler.NewHandlers(db.DB, redisClient, tushareClient, llmClient, hub, quoteCache, poolCache, cfg)

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
