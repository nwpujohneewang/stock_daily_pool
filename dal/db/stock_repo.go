package db

import (
	"context"

	"gorm.io/gorm/clause"
	"stock/internal/model"
)

var _ StockRepository = (*StockRepoImpl)(nil)

type StockRepository interface {
	GetByTsCode(ctx context.Context, tsCode string) (*model.StockBasicInfo, error)
	GetActiveStocks(ctx context.Context) ([]model.StockBasicInfo, error)
	Updates(ctx context.Context, stock *model.StockBasicInfo) error
	Upsert(ctx context.Context, stock *model.StockBasicInfo) error
	GetByBoardCode(ctx context.Context, boardCode string) ([]model.StockBasicInfo, error)
	Create(ctx context.Context, stock *model.StockBasicInfo) error
}

type StockRepoImpl struct{}

func NewStockRepository() *StockRepoImpl {
	return &StockRepoImpl{}
}

func (r StockRepoImpl) GetByTsCode(ctx context.Context, tsCode string) (*model.StockBasicInfo, error) {
	var stock model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).First(&stock).Error
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (r StockRepoImpl) GetActiveStocks(ctx context.Context) ([]model.StockBasicInfo, error) {
	var stocks []model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("status = ?", 1).Find(&stocks).Error
	return stocks, err
}

func (r StockRepoImpl) Upsert(ctx context.Context, stock *model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"symbol", "name", "exchange", "board_code",
			"industry", "is_st", "list_date", "status", "updated_at",
		}),
	}).Create(stock).Error
}

func (r StockRepoImpl) GetByBoardCode(ctx context.Context, boardCode string) ([]model.StockBasicInfo, error) {
	var stocks []model.StockBasicInfo
	err := PostgresStockDB(ctx).
		Where("board_code = ? AND status = ?", boardCode, 1).
		Find(&stocks).Error
	return stocks, err
}

func (r StockRepoImpl) Create(ctx context.Context, stock *model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Create(stock).Error
}

func (r StockRepoImpl) Updates(ctx context.Context, stock *model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Save(stock).Error
}
