package service

import (
	"context"
	"fmt"
	"log"
	"stock/dal/redis"
	"stock/model/dal_model"
	"time"

	"stock/dal/db"
)

type SnapshotService struct {
	logger     *log.Logger
	quoteCache redis.QuoteCacheInterface
}

func NewSnapshotService() *SnapshotService {
	return &SnapshotService{
		logger:     log.Default(),
		quoteCache: redis.NewQuoteCache(),
	}
}

type PoolStockItem struct {
	TsCode           string  `json:"ts_code"`
	Name             string  `json:"name"`
	ChangePct        float64 `json:"change_pct"`
	CurrentPrice     float64 `json:"current_price"`
	FirstLimitTime   string  `json:"first_limit_time,omitempty"`
	AttributionScore float64 `json:"attribution_score"`
}

type PoolGroup struct {
	TopicID   int64           `json:"topic_id"`
	TopicName string          `json:"topic_name"`
	Stocks    []PoolStockItem `json:"stocks"`
}

type PoolSnapshot struct {
	Date         string      `json:"date"`
	SnapshotTime string      `json:"snapshot_time"`
	LimitUp      PoolSection `json:"limit_up"`
	Above5Pct    PoolSection `json:"above_5pct"`
}

type PoolSection struct {
	TotalCount   int             `json:"total_count"`
	Groups       []PoolGroup     `json:"groups"`
	Unclassified []PoolStockItem `json:"unclassified"`
}

func (s *SnapshotService) TakeSnapshot(ctx context.Context, date string) error {
	poolCache := redis.NewPoolCache()
	limitUpCodes, err := poolCache.GetLimitUpMembers(ctx, date)
	if err != nil {
		return fmt.Errorf("get limit up pool: %w", err)
	}

	above5Codes, err := poolCache.GetAbove5Members(ctx, date)
	if err != nil {
		return fmt.Errorf("get above5 pool: %w", err)
	}

	t, _ := time.Parse("2006-01-02", date)
	stockRepo := db.NewStockRepository()
	poolRepo := db.NewPoolRepository()

	for _, tsCode := range limitUpCodes {
		stock, err := stockRepo.GetByTsCode(ctx, tsCode)
		if err != nil {
			s.logger.Printf("get stock %s failed: %v", tsCode, err)
			continue
		}

		quote, _ := s.quoteCache.Get(ctx, tsCode)
		record := &dal_model.DailyStockPool{
			Date:      t,
			TsCode:    tsCode,
			StockName: stock.Name,
			PoolType:  1,
		}
		if quote != nil {
			record.CurrentPrice = &quote.Price
			record.ChangePct = &quote.PctChg
			record.PreClose = &quote.PreClose
			vol := float64(quote.Vol)
			record.Vol = &vol
			record.Amount = &quote.Amount
		}
		if err := poolRepo.UpsertSnapshot(ctx, record); err != nil {
			s.logger.Printf("upsert limit up snapshot %s failed: %v", tsCode, err)
		}
	}

	for _, tsCode := range above5Codes {
		stock, err := stockRepo.GetByTsCode(ctx, tsCode)
		if err != nil {
			s.logger.Printf("get stock %s failed: %v", tsCode, err)
			continue
		}

		quote, _ := s.quoteCache.Get(ctx, tsCode)
		record := &dal_model.DailyStockPool{
			Date:      t,
			TsCode:    tsCode,
			StockName: stock.Name,
			PoolType:  2,
		}
		if quote != nil {
			record.CurrentPrice = &quote.Price
			record.ChangePct = &quote.PctChg
			record.PreClose = &quote.PreClose
			vol := float64(quote.Vol)
			record.Vol = &vol
			record.Amount = &quote.Amount
		}
		if err := poolRepo.UpsertSnapshot(ctx, record); err != nil {
			s.logger.Printf("upsert above5 snapshot %s failed: %v", tsCode, err)
		}
	}
	return nil
}

func (s *SnapshotService) GetSnapshot(ctx context.Context, date string) (*PoolSnapshot, error) {
	poolCache := redis.NewPoolCache()
	limitUpCodes, err := poolCache.GetLimitUpMembers(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("get limit up members: %w", err)
	}

	above5Codes, err := poolCache.GetAbove5Members(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("get above5 members: %w", err)
	}

	stockRepo := db.NewStockRepository()
	limitUpStocks := make([]PoolStockItem, 0, len(limitUpCodes))
	for _, tsCode := range limitUpCodes {
		stock, err := stockRepo.GetByTsCode(ctx, tsCode)
		if err != nil {
			continue
		}
		limitUpStocks = append(limitUpStocks, PoolStockItem{
			TsCode: tsCode,
			Name:   stock.Name,
		})
	}

	above5Stocks := make([]PoolStockItem, 0, len(above5Codes))
	for _, tsCode := range above5Codes {
		stock, err := stockRepo.GetByTsCode(ctx, tsCode)
		if err != nil {
			continue
		}
		above5Stocks = append(above5Stocks, PoolStockItem{
			TsCode: tsCode,
			Name:   stock.Name,
		})
	}

	return &PoolSnapshot{
		Date:         date,
		SnapshotTime: time.Now().Format("15:04:05"),
		LimitUp: PoolSection{
			TotalCount:   len(limitUpStocks),
			Groups:       []PoolGroup{},
			Unclassified: limitUpStocks,
		},
		Above5Pct: PoolSection{
			TotalCount:   len(above5Stocks),
			Groups:       []PoolGroup{},
			Unclassified: above5Stocks,
		},
	}, nil
}
