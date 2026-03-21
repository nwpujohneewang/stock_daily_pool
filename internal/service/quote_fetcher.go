package service

import (
	"context"
	"log"
	"stock/dal/redis"
	"sync"
	"time"

	"stock/dal/db"
	"stock/internal/external/tushare"
	"stock/internal/model"
	"stock/internal/pkg/shard"
)

type QuoteFetcher struct {
	tushareClient *tushare.Client
	shardCount    int
	logger        *log.Logger
}

func NewQuoteFetcher(
	tushareClient *tushare.Client,
	shardCount int,
) *QuoteFetcher {
	return &QuoteFetcher{
		tushareClient: tushareClient,
		shardCount:    shardCount,
		logger:        log.Default(),
	}
}

// FetchAllQuotes 在每个10s周期调用，将全市场行情写入 Redis rt:quote:{ts_code}
// 必须在 MonitorService.ProcessTick() 之前执行
func (f *QuoteFetcher) FetchAllQuotes(ctx context.Context) error {
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

func (f *QuoteFetcher) fetchBatch(ctx context.Context, tsCodes []string) {
	quotes, err := f.tushareClient.RealtimeQuote(ctx, tsCodes)
	if err != nil {
		f.logger.Printf("fetch quotes batch failed: %v", err)
		return
	}
	now := time.Now()
	quoteCache := redis.NewQuoteCache()
	for _, q := range quotes {
		stockQuote := &model.StockQuote{
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
			f.logger.Printf("set quote %s failed: %v", q.TsCode, err)
		}
	}
}
