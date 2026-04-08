package service

import (
	"context"
	"time"

	"stock/internal/service/classify"
	"stock/internal/service/crawler"
	"stock/internal/service/focus"
	"stock/internal/service/pool"
	"stock/internal/service/stock"
	"stock/internal/service/topic"
	"stock/model/dal_model"
)

// MonitorServiceInterface 监控服务接口
type MonitorServiceInterface interface {
	Start(ctx context.Context)
	ProcessTick(ctx context.Context, date string) error
	DetectBoard(tsCode string) dal_model.BoardCode
	SetNotifyCallback(fn func(date string))
	ResetClassifiedToday()
}

// ClassifyServiceInterface 分类服务接口
type ClassifyServiceInterface interface {
	ClassifyStock(ctx context.Context, tsCode string, date string, quoteTime time.Time) ([]dal_model.TopicRelation, error)
	ClassifyStockBatch(ctx context.Context, tsCodes []string, date string, quoteTime time.Time) (map[string][]dal_model.TopicRelation, error)
	ClassifyBySimpleHeat(ctx context.Context, stocks []classify.StockQuoteInput, date string) (map[string][]dal_model.TopicRelation, error)
	NormalizeTopicName(ctx context.Context, rawName string) (int64, string, bool, error)
}

// CrawlerServiceInterface 爬虫服务接口
type CrawlerServiceInterface interface {
	CrawlDate(ctx context.Context, date string) error
	CrawlHistory(ctx context.Context, startDate string) error
	CrawlToday(ctx context.Context) error
	CrawlWithParams(ctx context.Context, params crawler.CrawlParams) (*crawler.CrawlResult, error)
	ValidateTopics(ctx context.Context, topicNames []string) (missing []string, err error)
}

// SnapshotServiceInterface 快照服务接口
type SnapshotServiceInterface interface {
	TakeSnapshot(ctx context.Context, date string) error
}

// StockServiceInterface 股票服务接口
type StockServiceInterface interface {
	SyncStockBasic(ctx context.Context, date string) error
	GetStock(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error)
	GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error)
}

// StockQueryService 股票查询服务接口
type StockQueryService interface {
	Search(ctx context.Context, params stock.StockSearchParams) (*stock.StockSearchResult, error)
	GetDetail(ctx context.Context, tsCode string) (*stock.StockDetailResult, error)
}

// QuoteFetcherInterface 行情获取接口
type QuoteFetcherInterface interface {
	FetchAllQuotes(ctx context.Context) error
}

// TopicDictService 话题词典服务接口
type TopicDictService interface {
	List(ctx context.Context) ([]topic.TopicDictResult, error)
	Create(ctx context.Context, params topic.TopicDictParams) (*topic.TopicDictResult, error)
	Update(ctx context.Context, id int64, params topic.TopicDictParams) error
	Delete(ctx context.Context, id int64) error
}

// TopicQueryService 话题查询服务接口
type TopicQueryService interface {
	List(ctx context.Context, params topic.TopicQueryParams) (*topic.TopicListResult, error)
	GetByID(ctx context.Context, id int64) (*topic.TopicQueryResult, error)
}

// FocusService 关注话题服务接口
type FocusService interface {
	Get(ctx context.Context, date string) ([]focus.FocusTopicResult, error)
	Set(ctx context.Context, date string, topicIDs []int64) error
	Delete(ctx context.Context, date string, topicID int64) error
}

// PoolService 池服务接口
type PoolService interface {
	GetLimitUp(ctx context.Context, params pool.PoolQueryParams) ([]pool.PoolItemResult, error)
	GetAbove5(ctx context.Context, params pool.PoolQueryParams) ([]pool.PoolItemResult, error)
	Reclassify(ctx context.Context, date string) (*pool.ReclassifyResult, error)
}
