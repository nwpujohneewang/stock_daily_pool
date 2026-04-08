package redis

import (
	"context"
	"fmt"
	"stock/model/dal_model"
	"time"
)

type QuoteCacheInterface interface {
	Get(ctx context.Context, tsCode string) (*dal_model.StockQuote, error)
	Set(ctx context.Context, quote *dal_model.StockQuote) error
	Delete(ctx context.Context, tsCode string) error
	GetAll(ctx context.Context) ([]*dal_model.StockQuote, error)
}

var _ QuoteCacheInterface = (*QuoteCacheImpl)(nil)

type QuoteCacheImpl struct{}

func NewQuoteCache() *QuoteCacheImpl {
	return &QuoteCacheImpl{}
}

var shanghaiLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}()

func nowShanghai() time.Time {
	return time.Now().In(shanghaiLoc)
}

func (c QuoteCacheImpl) Get(ctx context.Context, tsCode string) (*dal_model.StockQuote, error) {
	key := fmt.Sprintf("rt:quote:%s", tsCode)
	data, err := RedisClient(ctx).HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	quote := &dal_model.StockQuote{TsCode: tsCode}
	if v, ok := data["price"]; ok {
		fmt.Sscanf(v, "%lf", &quote.Price)
	}
	if v, ok := data["pre_close"]; ok {
		fmt.Sscanf(v, "%lf", &quote.PreClose)
	}
	if v, ok := data["pct_chg"]; ok {
		fmt.Sscanf(v, "%lf", &quote.PctChg)
	}
	if v, ok := data["vol"]; ok {
		fmt.Sscanf(v, "%d", &quote.Vol)
	}
	if v, ok := data["amount"]; ok {
		fmt.Sscanf(v, "%lf", &quote.Amount)
	}
	if v, ok := data["turnover"]; ok {
		fmt.Sscanf(v, "%lf", &quote.TurnoverRate)
	}
	if v, ok := data["update_time"]; ok {
		if t, err := time.Parse("15:04:05", v); err == nil {
			now := nowShanghai()
			quote.UpdateTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, shanghaiLoc)
		}
	}

	return quote, nil
}

func (c QuoteCacheImpl) Set(ctx context.Context, quote *dal_model.StockQuote) error {
	key := fmt.Sprintf("rt:quote:%s", quote.TsCode)

	fields := map[string]interface{}{
		"price":       fmt.Sprintf("%.2f", quote.Price),
		"pre_close":   fmt.Sprintf("%.2f", quote.PreClose),
		"pct_chg":     fmt.Sprintf("%.2f", quote.PctChg),
		"vol":         quote.Vol,
		"amount":      fmt.Sprintf("%.2f", quote.Amount),
		"turnover":    fmt.Sprintf("%.2f", quote.TurnoverRate),
		"update_time": quote.UpdateTime.Format("15:04:05"),
	}

	return RedisClient(ctx).HSet(ctx, key, fields).Err()
}

func (c QuoteCacheImpl) Delete(ctx context.Context, tsCode string) error {
	key := fmt.Sprintf("rt:quote:%s", tsCode)
	return RedisClient(ctx).Del(ctx, key).Err()
}

func (c QuoteCacheImpl) GetAll(ctx context.Context) ([]*dal_model.StockQuote, error) {
	var quotes []*dal_model.StockQuote
	var cursor uint64
	pattern := "rt:quote:*"

	for {
		keys, nextCursor, err := RedisClient(ctx).Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			// Extract ts_code from key "rt:quote:{ts_code}"
			tsCode := key[9:] // len("rt:quote:") = 9

			quote, err := c.Get(ctx, tsCode)
			if err != nil {
				continue
			}
			if quote != nil {
				quotes = append(quotes, quote)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return quotes, nil
}
