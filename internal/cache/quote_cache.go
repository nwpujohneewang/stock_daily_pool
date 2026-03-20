package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"stock/internal/model"
)

type QuoteCache struct {
	client *redis.Client
}

func NewQuoteCache(client *redis.Client) *QuoteCache {
	return &QuoteCache{client: client}
}

func (c *QuoteCache) Get(ctx context.Context, tsCode string) (*model.StockQuote, error) {
	key := fmt.Sprintf("rt:quote:%s", tsCode)
	data, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	quote := &model.StockQuote{TsCode: tsCode}
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
			now := time.Now()
			quote.UpdateTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
		}
	}

	return quote, nil
}

func (c *QuoteCache) Set(ctx context.Context, quote *model.StockQuote) error {
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

	return c.client.HSet(ctx, key, fields).Err()
}

func (c *QuoteCache) Delete(ctx context.Context, tsCode string) error {
	key := fmt.Sprintf("rt:quote:%s", tsCode)
	return c.client.Del(ctx, key).Err()
}
