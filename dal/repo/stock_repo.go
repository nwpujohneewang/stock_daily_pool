package repo

import (
	"context"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/external/tushare"
	"stock/model/dal_model"
)

type StockRepository interface {
	GetAllStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error)
	GetByTsCode(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error)
	GetByTsCodes(ctx context.Context, tsCodes []string) ([]dal_model.StockBasicInfo, error)
	GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error)
	Updates(ctx context.Context, stock *dal_model.StockBasicInfo) error
	Upsert(ctx context.Context, stock *dal_model.StockBasicInfo) error
	UpsertBatch(ctx context.Context, stocks []dal_model.StockBasicInfo) error
	SetSTBatch(ctx context.Context, tsCodes []string, isST bool) error
	GetByBoardCode(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error)
	Create(ctx context.Context, stock *dal_model.StockBasicInfo) error
	Search(ctx context.Context, query string, limit int) ([]dal_model.StockBasicInfo, error)
	GetPaginatedStocks(ctx context.Context, query, topic, category string, page, pageSize int) ([]dal_model.StockBasicInfo, int64, error)
	UpdateMarketValueBatch(ctx context.Context, items []tushare.DailyBasicItem) error
}

type stockRepoImpl struct {
	dao dao.StockDAO
}

func NewStockRepository() StockRepository {
	return &stockRepoImpl{dao: dao.NewStockDAO()}
}

func (r *stockRepoImpl) GetAllStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	stockCache := cache.NewStockCache()
	if stocks, found := stockCache.GetAllStocks(ctx); found {
		return stocks, nil
	}
	stocks, err := r.dao.GetAllStocks(ctx)
	if err != nil {
		return nil, err
	}
	_ = stockCache.SetAllStocks(ctx, stocks)
	return stocks, nil
}

func (r *stockRepoImpl) GetByTsCode(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error) {
	stockCache := cache.NewStockCache()
	if stockMap, found := stockCache.GetAllStocksMap(ctx); found {
		if stock, ok := stockMap[tsCode]; ok {
			copied := stock
			return &copied, nil
		}
	}
	return r.dao.GetByTsCode(ctx, tsCode)
}

func (r *stockRepoImpl) GetByTsCodes(ctx context.Context, tsCodes []string) ([]dal_model.StockBasicInfo, error) {
	stockCache := cache.NewStockCache()
	if stockMap, found := stockCache.GetAllStocksMap(ctx); found {
		stocks := make([]dal_model.StockBasicInfo, 0, len(tsCodes))
		missing := make([]string, 0)
		for _, tsCode := range tsCodes {
			if stock, ok := stockMap[tsCode]; ok {
				stocks = append(stocks, stock)
			} else {
				missing = append(missing, tsCode)
			}
		}
		if len(missing) == 0 {
			return stocks, nil
		}
		missingStocks, err := r.dao.GetByTsCodes(ctx, missing)
		if err != nil {
			return nil, err
		}
		return append(stocks, missingStocks...), nil
	}
	return r.dao.GetByTsCodes(ctx, tsCodes)
}

func (r *stockRepoImpl) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	stockCache := cache.NewStockCache()
	if stocks, found := stockCache.GetActiveStocks(ctx); found {
		return stocks, nil
	}
	stocks, err := r.dao.GetActiveStocks(ctx)
	if err != nil {
		return nil, err
	}
	_ = stockCache.SetActiveStocks(ctx, stocks)
	return stocks, nil
}

func (r *stockRepoImpl) Updates(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	if err := r.dao.Updates(ctx, stock); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *stockRepoImpl) Upsert(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	if err := r.dao.Upsert(ctx, stock); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *stockRepoImpl) UpsertBatch(ctx context.Context, stocks []dal_model.StockBasicInfo) error {
	if err := r.dao.UpsertBatch(ctx, stocks); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *stockRepoImpl) SetSTBatch(ctx context.Context, tsCodes []string, isST bool) error {
	if err := r.dao.SetSTBatch(ctx, tsCodes, isST); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *stockRepoImpl) GetByBoardCode(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error) {
	return r.dao.GetByBoardCode(ctx, boardCode)
}

func (r *stockRepoImpl) Create(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	if err := r.dao.Create(ctx, stock); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *stockRepoImpl) Search(ctx context.Context, query string, limit int) ([]dal_model.StockBasicInfo, error) {
	return r.dao.Search(ctx, query, limit)
}

func (r *stockRepoImpl) GetPaginatedStocks(ctx context.Context, query, topic, category string, page, pageSize int) ([]dal_model.StockBasicInfo, int64, error) {
	return r.dao.GetPaginatedStocks(ctx, query, topic, category, page, pageSize)
}

func (r *stockRepoImpl) UpdateMarketValueBatch(ctx context.Context, items []tushare.DailyBasicItem) error {
	if err := r.dao.UpdateMarketValueBatch(ctx, items); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *stockRepoImpl) invalidateCaches(ctx context.Context) error {
	stockCache := cache.NewStockCache()
	_ = stockCache.ClearActiveStocks(ctx)
	_ = stockCache.ClearAllStocks(ctx)
	return nil
}
