package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"stock/config"
	"stock/internal/external/tushare"
	"stock/internal/model"
	"stock/internal/pkg/limiter"
	"stock/internal/repo"
)

type StockService struct {
	stockRepo     *repo.StockRepo
	boardRepo     *repo.BoardRepo
	tushareClient *tushare.Client
	cfg           *config.TushareConfig
	logger        *log.Logger
}

func NewStockService(
	stockRepo *repo.StockRepo,
	boardRepo *repo.BoardRepo,
	tushareClient *tushare.Client,
	cfg *config.TushareConfig,
) *StockService {
	return &StockService{
		stockRepo:     stockRepo,
		boardRepo:     boardRepo,
		tushareClient: tushareClient,
		cfg:           cfg,
		logger:        log.Default(),
	}
}

func (s *StockService) SyncStockBasic(ctx context.Context) error {
	stocks, err := s.tushareClient.StockBasic(ctx)
	if err != nil {
		return fmt.Errorf("fetch stock basic: %w", err)
	}

	for _, stock := range stocks {
		boardCode := s.detectBoard(stock.Symbol)
		industry := stock.Industry
		listDate, _ := time.Parse("2006-01-02", stock.ListDate)
		bs := &model.StockBasicInfo{
			TsCode:    stock.TsCode,
			Symbol:    stock.Symbol,
			Name:      stock.Name,
			Exchange:  stock.Exchange,
			BoardCode: boardCode,
			Industry:  &industry,
			IsST:      stock.IsST,
			ListDate:  &listDate,
			Status:    1,
		}
		if err := s.stockRepo.Upsert(ctx, bs); err != nil {
			s.logger.Printf("upsert stock %s failed: %v", stock.TsCode, err)
		}
	}
	return nil
}

func (s *StockService) detectBoard(symbol string) string {
	return string(limiter.DetectBoard(symbol))
}

func (s *StockService) GetStock(ctx context.Context, tsCode string) (*model.StockBasicInfo, error) {
	return s.stockRepo.GetByTsCode(ctx, tsCode)
}

func (s *StockService) GetActiveStocks(ctx context.Context) ([]model.StockBasicInfo, error) {
	return s.stockRepo.GetActiveStocks(ctx)
}

func (s *StockService) GetStocksByBoard(ctx context.Context, boardCode string) ([]model.StockBasicInfo, error) {
	return s.stockRepo.GetByBoardCode(ctx, boardCode)
}

func (s *StockService) LoadBoardRules(ctx context.Context) (map[model.BoardCode]*model.BoardRule, error) {
	boards, err := s.boardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all boards: %w", err)
	}

	result := make(map[model.BoardCode]*model.BoardRule)
	for i := range boards {
		result[boards[i].BoardCode] = &boards[i]
	}
	return result, nil
}
