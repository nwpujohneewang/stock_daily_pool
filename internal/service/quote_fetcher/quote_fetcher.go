package quote_fetcher

import (
	"context"
	"stock/external/tushare"
	"time"

	"stock/dal/cache"
	"stock/internal/pkg/logger"
	"stock/model/dal_model"

	"go.uber.org/zap"
)

type QuoteFetcherImpl struct {
	tushareClient *tushare.Client
}

func NewQuoteFetcher(tushareClient *tushare.Client) *QuoteFetcherImpl {
	return &QuoteFetcherImpl{
		tushareClient: tushareClient,
	}
}

// FetchAllQuotes fetches all market quotes via rt_k API, caches stocks with pct_chg >= 5% or in yesterday strong pool, and returns them
func (f *QuoteFetcherImpl) FetchAllQuotes(ctx context.Context) ([]*dal_model.StockQuote, error) {
	quotes, err := f.tushareClient.RealtimeQuoteAll(ctx)

	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Now().In(loc).Format("2006-01-02")

	poolCache := cache.NewPoolCache()
	quoteCache := cache.NewQuoteCache()
	now := time.Now()
	var result []*dal_model.StockQuote

	for _, q := range quotes {
		shouldCache := q.PctChg >= 5.0 || poolCache.IsYesterdayStrong(ctx, today, q.TsCode)

		if !shouldCache {
			continue
		}

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

		result = append(result, stockQuote)
	}

	if err := quoteCache.SetBatch(ctx, result); err != nil {
		logger.Error("set quote batch failed", zap.Error(err))
		return nil, err
	}

	return result, nil
}
