package service

import (
	"context"
	"go.uber.org/zap"
	"stock/dal/redis"
	"stock/internal/pkg/logger"
	"stock/model/dal_model"
	"sync"
	"time"

	"stock/dal/db"
	"stock/internal/external/tushare"
	"stock/internal/pkg/shard"
)

type QuoteFetcherImpl struct {
	tushareClient *tushare.Client
	shardCount    int
}

var _ QuoteFetcherInterface = (*QuoteFetcherImpl)(nil)

func NewQuoteFetcher(
	tushareClient *tushare.Client,
	shardCount int,
) *QuoteFetcherImpl {
	return &QuoteFetcherImpl{
		tushareClient: tushareClient,
		shardCount:    shardCount,
	}
}

// FetchAllQuotes 在每个10s周期调用，将全市场行情写入 Redis rt:quote:{ts_code}
// 必须在 MonitorService.ProcessTick() 之前执行
func (f *QuoteFetcherImpl) FetchAllQuotes(ctx context.Context) error {
	stockRepo := db.NewStockRepository()
	stocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		return err
	}

	tsCodes := make([]string, len(stocks))
	for i, s := range stocks {
		tsCodes[i] = s.TsCode
	}

	batches := shard.BuildShardBatches(tsCodes, f.shardCount)

	var wg sync.WaitGroup
	for _, batch := range batches {
		wg.Add(1)
		go func(batch shard.ShardBatch) {
			defer wg.Done()
			f.fetchBatch(ctx, batch.TsCodes)
		}(batch)
	}
	wg.Wait()
	return nil
}

func (f *QuoteFetcherImpl) fetchBatch(ctx context.Context, tsCodes []string) {
	quotes, err := f.tushareClient.RealtimeQuote(ctx, tsCodes)
	if err != nil {
		logger.Error("fetch quotes batch failed: %v", zap.Error(err))
		return
	}
	now := time.Now()
	quoteCache := redis.NewQuoteCache()
	for _, q := range quotes {
		stockQuote := &dal_model.StockQuote{
			TsCode:       q.TsCode,
			PreClose:     q.PreClose,
			Price:        q.Price,
			PctChg:       q.PctChg,
			Vol:          q.Vol,
			Amount:       q.Amount,
			TurnoverRate: q.TurnoverRate,
			UpdateTime:   now,
		}
		if err := quoteCache.Set(ctx, stockQuote); err != nil {
			logger.Error("set quote failed", zap.String("code", q.TsCode), zap.Error(err))
		}
	}
}
