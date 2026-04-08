package cache

import (
	"context"
	"stock/model/dal_model"
)

const (
	activeStocksKey = "stock:active:status=1"
	allStocksKey    = "stock:all"
)

type StockCacheInterface interface {
	SetActiveStocks(ctx context.Context, stocks []dal_model.StockBasicInfo) error
	GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, bool)
	ClearActiveStocks(ctx context.Context) error
	SetAllStocks(ctx context.Context, stocks []dal_model.StockBasicInfo) error
	GetAllStocks(ctx context.Context) ([]dal_model.StockBasicInfo, bool)
	GetAllStocksMap(ctx context.Context) (map[string]dal_model.StockBasicInfo, bool)
	ClearAllStocks(ctx context.Context) error
}

type StockCacheImpl struct{}

func NewStockCache() *StockCacheImpl {
	return &StockCacheImpl{}
}

func (c StockCacheImpl) SetActiveStocks(ctx context.Context, stocks []dal_model.StockBasicInfo) error {
	Cache.Set(activeStocksKey, stocks, TTLUntilEndOfDay())
	return nil
}

func (c StockCacheImpl) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, bool) {
	if v, found := Cache.Get(activeStocksKey); found {
		if stocks, ok := v.([]dal_model.StockBasicInfo); ok {
			return stocks, true
		}
	}
	return nil, false
}

func (c StockCacheImpl) ClearActiveStocks(ctx context.Context) error {
	Cache.Delete(activeStocksKey)
	return nil
}

func (c StockCacheImpl) SetAllStocks(ctx context.Context, stocks []dal_model.StockBasicInfo) error {
	stockMap := make(map[string]dal_model.StockBasicInfo, len(stocks))
	for _, stock := range stocks {
		stockMap[stock.TsCode] = stock
	}
	Cache.Set(allStocksKey, stockMap, TTLUntilEndOfDay())
	return nil
}

func (c StockCacheImpl) GetAllStocks(ctx context.Context) ([]dal_model.StockBasicInfo, bool) {
	if v, found := Cache.Get(allStocksKey); found {
		if stockMap, ok := v.(map[string]dal_model.StockBasicInfo); ok {
			stocks := make([]dal_model.StockBasicInfo, 0, len(stockMap))
			for _, stock := range stockMap {
				stocks = append(stocks, stock)
			}
			return stocks, true
		}
	}
	return nil, false
}

func (c StockCacheImpl) GetAllStocksMap(ctx context.Context) (map[string]dal_model.StockBasicInfo, bool) {
	if v, found := Cache.Get(allStocksKey); found {
		if stockMap, ok := v.(map[string]dal_model.StockBasicInfo); ok {
			copied := make(map[string]dal_model.StockBasicInfo, len(stockMap))
			for tsCode, stock := range stockMap {
				copied[tsCode] = stock
			}
			return copied, true
		}
	}
	return nil, false
}

func (c StockCacheImpl) ClearAllStocks(ctx context.Context) error {
	Cache.Delete(allStocksKey)
	return nil
}
