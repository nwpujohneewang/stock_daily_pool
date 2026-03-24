package service

import (
	"context"
	"stock/model/dal_model"
	"time"
)

type MonitorServiceInterface interface {
	Start(ctx context.Context)
	ProcessTick(ctx context.Context, date string) error
	DetectBoard(tsCode string) dal_model.BoardCode
}

type ClassifyServiceInterface interface {
	ClassifyStock(ctx context.Context, tsCode string, date string, quoteTime time.Time) ([]dal_model.TopicRelation, error)
	ClassifyStockBatch(ctx context.Context, tsCodes []string, date string, quoteTime time.Time) (map[string][]dal_model.TopicRelation, error)
	NormalizeTopicName(ctx context.Context, rawName string) (int64, string, bool, error)
}

type AlertServiceInterface interface {
	CheckAndAlert(ctx context.Context, input AlertCheckInput) (*AlertCheckOutput, error)
	GetTodayAlerts(ctx context.Context, date string) ([]dal_model.StrategyAlert, error)
	GetHistoryAlerts(ctx context.Context, startDate, endDate string, topicID *int64, page, pageSize int) ([]dal_model.StrategyAlert, int64, error)
}

type CrawlerServiceInterface interface {
	CrawlDate(ctx context.Context, date string) error
	CrawlHistory(ctx context.Context, startDate string) error
	CrawlToday(ctx context.Context) error
}

type ConceptSyncServiceInterface interface {
	SyncAll(ctx context.Context) error
	SyncIncremental(ctx context.Context) error
	BuildMappings(ctx context.Context) error
	GetUnmappedConcepts(ctx context.Context) ([]string, error)
}

type SnapshotServiceInterface interface {
	TakeSnapshot(ctx context.Context, date string) error
	GetSnapshot(ctx context.Context, date string) (*PoolSnapshot, error)
}

type StockServiceInterface interface {
	SyncStockBasic(ctx context.Context) error
	GetStock(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error)
	GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error)
	GetStocksByBoard(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error)
	LoadBoardRules(ctx context.Context) (map[dal_model.BoardCode]*dal_model.BoardRule, error)
}

type QuoteFetcherInterface interface {
	FetchAllQuotes(ctx context.Context) error
}
