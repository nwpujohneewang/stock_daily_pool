package handler

import (
	"stock/config"
	"stock/external/news"
	"stock/internal/scheduler"
)

type Handlers struct {
	Config *config.AppConfig

	PoolHandler
	TopicHandler
	FocusHandler
	CrawlHandler
	TopicDictionaryHandler
	StockHandler
	StockTopicRelationHandler
	MonitorHandler
	SnapshotHandler
	SchedulerHandler
	LLMClassifyHandler
	HotSpotHandler
}

func NewHandlers(
	cfg *config.AppConfig,
	s *scheduler.Scheduler,
	newsAgg *news.Aggregator,
) *Handlers {
	h := &Handlers{
		Config: cfg,
	}

	h.PoolHandler = *NewPoolHandler()
	h.TopicHandler = *NewTopicHandler()
	h.FocusHandler = *NewFocusHandler()
	h.CrawlHandler = *NewCrawlHandler()
	h.TopicDictionaryHandler = *NewTopicDictionaryHandler()
	h.StockHandler = *NewStockHandler()
	h.StockTopicRelationHandler = *NewStockTopicRelationHandler()
	h.MonitorHandler = *NewMonitorHandler()
	h.SnapshotHandler = *NewSnapshotHandler()
	h.SchedulerHandler = *NewSchedulerHandler(s)
	h.LLMClassifyHandler = *NewLLMClassifyHandler()
	h.HotSpotHandler = *NewHotSpotHandler(newsAgg)

	return h
}
