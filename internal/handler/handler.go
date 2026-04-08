package handler

import (
	"stock/config"
	"stock/internal/scheduler"
	"stock/internal/ws"
)

type Handlers struct {
	Hub    *ws.Hub
	Config *config.AppConfig

	PoolHandler
	TopicHandler
	FocusHandler
	WSHandler
	CrawlHandler
	TopicDictionaryHandler
	StockHandler
	StockTopicRelationHandler
	MonitorHandler
	SnapshotHandler
	SchedulerHandler
}

func NewHandlers(
	hub *ws.Hub,
	cfg *config.AppConfig,
	s *scheduler.Scheduler,
) *Handlers {
	h := &Handlers{
		Hub:    hub,
		Config: cfg,
	}

	h.PoolHandler = *NewPoolHandler()
	h.TopicHandler = *NewTopicHandler()
	h.FocusHandler = *NewFocusHandler()
	h.WSHandler = *NewWSHandler(hub)
	h.CrawlHandler = *NewCrawlHandler()
	h.TopicDictionaryHandler = *NewTopicDictionaryHandler()
	h.StockHandler = *NewStockHandler()
	h.StockTopicRelationHandler = *NewStockTopicRelationHandler()
	h.MonitorHandler = *NewMonitorHandler()
	h.SnapshotHandler = *NewSnapshotHandler()
	h.SchedulerHandler = *NewSchedulerHandler(s)

	return h
}
