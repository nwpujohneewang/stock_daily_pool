package db

import (
	"context"
	"stock/model/dal_model"

	"gorm.io/gorm/clause"
)

var _ StockRepository = (*StockRepoImpl)(nil)

type StockRepository interface {
	GetByTsCode(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error)
	GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error)
	Updates(ctx context.Context, stock *dal_model.StockBasicInfo) error
	Upsert(ctx context.Context, stock *dal_model.StockBasicInfo) error
	UpsertBatch(ctx context.Context, stocks []dal_model.StockBasicInfo) error
	SetSTBatch(ctx context.Context, tsCodes []string, isST bool) error
	GetByBoardCode(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error)
	Create(ctx context.Context, stock *dal_model.StockBasicInfo) error
}

type StockRepoImpl struct{}

func NewStockRepository() *StockRepoImpl {
	return &StockRepoImpl{}
}

func (r StockRepoImpl) GetByTsCode(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error) {
	var stock dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).First(&stock).Error
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (r StockRepoImpl) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	var stocks []dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("status = ?", 1).Find(&stocks).Error
	return stocks, err
}

func (r StockRepoImpl) Upsert(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"symbol", "name", "exchange", "board_code",
			"industry", "is_st", "list_date", "status", "updated_at",
		}),
	}).Create(stock).Error
}

func (r StockRepoImpl) UpsertBatch(ctx context.Context, stocks []dal_model.StockBasicInfo) error {
	if len(stocks) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"symbol", "name", "exchange", "board_code",
			"industry", "is_st", "list_date", "status", "updated_at",
		}),
	}).CreateInBatches(&stocks, 500).Error
}

func (r StockRepoImpl) SetSTBatch(ctx context.Context, tsCodes []string, isST bool) error {
	if len(tsCodes) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).
		Model(&dal_model.StockBasicInfo{}).
		Where("ts_code IN ?", tsCodes).
		Update("is_st", isST).Error
}

func (r StockRepoImpl) GetByBoardCode(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error) {
	var stocks []dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).
		Where("board_code = ? AND status = ?", boardCode, 1).
		Find(&stocks).Error
	return stocks, err
}

func (r StockRepoImpl) Create(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Create(stock).Error
}

func (r StockRepoImpl) Updates(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Save(stock).Error
}
