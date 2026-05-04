package stock

import (
	"context"
	"fmt"
	"stock/external/tushare"
	"strings"
	"time"

	"stock/dal/repo"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/model/dal_model"

	"go.uber.org/zap"
)

type activeStockProvider interface {
	GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error)
}

type StockServiceImpl struct {
	tushareClient       *tushare.Client
	activeStockProvider activeStockProvider
}

func NewStockService(tushareClient *tushare.Client) *StockServiceImpl {
	return &StockServiceImpl{
		tushareClient: tushareClient,
	}
}

func (s *StockServiceImpl) SyncStockBasic(ctx context.Context, date string) error {
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

	dailyBasicMap := make(map[string]tushare.DailyBasicItem)
	if date != "" {
		tradeDate := strings.ReplaceAll(date, "-", "")
		basicItems, err := s.tushareClient.DailyBasic(ctx, tradeDate)
		if err != nil {
			logger.Warn("fetch daily_basic failed", zap.Error(err))
		} else {
			for _, item := range basicItems {
				dailyBasicMap[item.TsCode] = item
			}
			logger.Info("loaded daily_basic market value", zap.Int("count", len(basicItems)))
		}
	}

	stockRepo := repo.NewStockRepository()
	records := s.buildSyncStockRecords(tushareStocks, stMap, dailyBasicMap)
	if err = stockRepo.UpsertBatch(ctx, records); err != nil {
		return fmt.Errorf("upsert batch: %w", err)
	}

	return nil
}

func (s *StockServiceImpl) buildSyncStockRecords(
	tushareStocks []tushare.StockBasicItem,
	stMap map[string]bool,
	dailyBasicMap map[string]tushare.DailyBasicItem,
) []dal_model.StockBasicInfo {
	records := make([]dal_model.StockBasicInfo, 0, len(tushareStocks))
	for _, stock := range tushareStocks {
		boardCode := s.detectBoard(stock.Symbol)
		industry := stock.Industry
		listDate, _ := time.Parse("2006-01-02", stock.ListDate)
		record := dal_model.StockBasicInfo{
			TsCode:    stock.TsCode,
			Symbol:    stock.Symbol,
			Name:      stock.Name,
			Exchange:  stock.Exchange,
			BoardCode: boardCode,
			Industry:  &industry,
			IsST:      stMap[stock.TsCode],
			ListDate:  &listDate,
			Status:    1,
		}
		if dailyBasic, ok := dailyBasicMap[stock.TsCode]; ok {
			totalMv := dailyBasic.TotalMv
			circMv := dailyBasic.CircMv
			record.TotalMv = &totalMv
			record.CircMv = &circMv
		}
		records = append(records, record)
	}
	return records
}

func (s *StockServiceImpl) detectBoard(symbol string) string {
	return string(limiter.DetectBoard(symbol))
}

func (s *StockServiceImpl) GetStock(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error) {
	return repo.NewStockRepository().GetByTsCode(ctx, tsCode)
}

func (s *StockServiceImpl) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	provider := s.activeStockProvider
	if provider == nil {
		provider = repo.NewStockRepository()
	}
	return provider.GetActiveStocks(ctx)
}
