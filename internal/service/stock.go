package service

import (
	"context"
	"fmt"
	"stock/model/dal_model"
	"time"

	"stock/dal/db"
	"stock/internal/external/tushare"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"

	"go.uber.org/zap"
)

type StockServiceImpl struct {
	tushareClient *tushare.Client
}

var _ StockServiceInterface = (*StockServiceImpl)(nil)

func NewStockService(tushareClient *tushare.Client) *StockServiceImpl {
	return &StockServiceImpl{
		tushareClient: tushareClient,
	}
}

func (s *StockServiceImpl) SyncStockBasic(ctx context.Context) error {
	tushareStocks, err := s.tushareClient.StockBasic(ctx)
	if err != nil {
		return fmt.Errorf("fetch stock basic: %w", err)
	}

	stMap, err := s.tushareClient.StockST(ctx)
	if err != nil {
		logger.Warn("fetch stock_st failed, proceeding without ST info", zap.Error(err))
		stMap = make(map[string]bool)
	}

	logger.Info("共获取股票信息",
		zap.Int("total", len(tushareStocks)),
		zap.Int("st_count", len(stMap)))

	stockRepo := db.NewStockRepository()
	records := make([]dal_model.StockBasicInfo, 0, len(tushareStocks))
	for _, stock := range tushareStocks {
		boardCode := s.detectBoard(stock.Symbol)
		industry := stock.Industry
		listDate, _ := time.Parse("2006-01-02", stock.ListDate)
		records = append(records, dal_model.StockBasicInfo{
			TsCode:    stock.TsCode,
			Symbol:    stock.Symbol,
			Name:      stock.Name,
			Exchange:  stock.Exchange,
			BoardCode: boardCode,
			Industry:  &industry,
			IsST:      stMap[stock.TsCode],
			ListDate:  &listDate,
			Status:    1,
		})
	}

	if err = stockRepo.UpsertBatch(ctx, records); err != nil {
		return fmt.Errorf("upsert batch: %w", err)
	}

	if len(stMap) > 0 {
		stCodes := make([]string, 0, len(stMap))
		for tsCode := range stMap {
			stCodes = append(stCodes, tsCode)
		}
		if err = stockRepo.SetSTBatch(ctx, stCodes, true); err != nil {
			logger.Warn("set ST batch failed", zap.Error(err))
		}
	}

	return nil
}

func (s *StockServiceImpl) detectBoard(symbol string) string {
	return string(limiter.DetectBoard(symbol))
}

func (s *StockServiceImpl) GetStock(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error) {
	return db.NewStockRepository().GetByTsCode(ctx, tsCode)
}

func (s *StockServiceImpl) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	return db.NewStockRepository().GetActiveStocks(ctx)
}

func (s *StockServiceImpl) GetStocksByBoard(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error) {
	return db.NewStockRepository().GetByBoardCode(ctx, boardCode)
}

func (s *StockServiceImpl) LoadBoardRules(ctx context.Context) (map[dal_model.BoardCode]*dal_model.BoardRule, error) {
	boardRepo := db.NewBoardRepository()
	boards, err := boardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all boards: %w", err)
	}

	result := make(map[dal_model.BoardCode]*dal_model.BoardRule)
	for i := range boards {
		result[boards[i].BoardCode] = &boards[i]
	}
	return result, nil
}
