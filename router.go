package main

import (
	"net/http"
	"stock/config"
	"stock/internal/handler"
	"stock/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRouter(cfg *config.AppConfig, h *handler.Handlers) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.Use(middleware.IpLogger())

	// Serve frontend static files
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/favicon.svg", "./frontend/dist/favicon.svg")
	r.StaticFile("/icons.svg", "./frontend/dist/icons.svg")

	// Serve index.html for root path
	r.GET("/", func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(cfg.Server.APIKey))
	{
		pool := api.Group("/pool")
		{
			pool.GET("/reclassify", h.PoolHandler.ReclassifyByDate)
		}

		topics := api.Group("/topics")
		{
			topics.GET("", h.TopicHandler.List)
		}

		focus := api.Group("/focus")
		{
			focus.GET("", h.FocusHandler.Get)
			focus.POST("", h.FocusHandler.Set)
			focus.POST("/delete/:topic_id", h.FocusHandler.Delete)
		}

		crawl := api.Group("/crawl")
		{
			crawl.POST("/jiuyan", h.CrawlHandler.CrawlJiuyan)
			crawl.POST("/rebuild", h.CrawlHandler.RebuildTopicRelations)
		}

		dict := api.Group("/topic-dictionary")
		{
			dict.GET("", h.TopicDictionaryHandler.List)
			dict.POST("", h.TopicDictionaryHandler.Create)
			dict.POST("/update/:id", h.TopicDictionaryHandler.Update)
			dict.POST("/delete/:id", h.TopicDictionaryHandler.Delete)
		}

		stocks := api.Group("/stocks")
		{
			stocks.GET("/search", h.StockHandler.Search)
			stocks.GET("/:ts_code", h.StockHandler.GetDetail)
			stocks.POST("/sync_basic", h.StockHandler.SyncBasic)
			stocks.POST("/:ts_code/topic-relations/manual", h.StockTopicRelationHandler.AddManual)
			stocks.DELETE("/:ts_code/topic-relations/:topic_id", h.StockTopicRelationHandler.Delete)
		}

		monitorGroup := api.Group("/monitor")
		{
			monitorGroup.POST("/start", h.MonitorHandler.Start)
			monitorGroup.POST("/stop", h.MonitorHandler.Stop)
			monitorGroup.POST("/tick", h.MonitorHandler.Tick)
			monitorGroup.GET("/status", h.MonitorHandler.Status)
		}

		snapshotGroup := api.Group("/snapshot")
		{
			snapshotGroup.POST("/trigger", h.SnapshotHandler.TriggerSnapshot)
			snapshotGroup.POST("/trigger-range", h.SnapshotHandler.TriggerSnapshotRange)
		}

		schedulerGroup := api.Group("/scheduler")
		{
			schedulerGroup.POST("/pre_market_init", h.SchedulerHandler.TriggerPreMarketInit)
			schedulerGroup.POST("/load_yesterday_strong", h.SchedulerHandler.TriggerLoadYesterdayStrongPool)
			schedulerGroup.POST("/closing_snapshot", h.SchedulerHandler.TriggerClosingSnapshot)
			schedulerGroup.POST("/sync_stock_basic", h.SchedulerHandler.TriggerSyncStockBasic)
		}

		llmClassify := api.Group("/llm-classify")
		{
			llmClassify.POST("/run-history", h.LLMClassifyHandler.RunHistory)
		}

		hotSpot := api.Group("/hot-spot")
		{
			hotSpot.GET("/fetch-news", h.HotSpotHandler.FetchNews)
		}
	}

	return r
}
