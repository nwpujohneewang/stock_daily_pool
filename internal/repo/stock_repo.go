package repo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"stock/internal/model"
)

type StockRepo struct {
	db *gorm.DB
}

func NewStockRepo(db *gorm.DB) *StockRepo {
	return &StockRepo{db: db}
}

func (r *StockRepo) GetByTsCode(ctx context.Context, tsCode string) (*model.StockBasicInfo, error) {
	var stock model.StockBasicInfo
	err := r.db.WithContext(ctx).Where("ts_code = ?", tsCode).First(&stock).Error
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (r *StockRepo) GetActiveStocks(ctx context.Context) ([]model.StockBasicInfo, error) {
	var stocks []model.StockBasicInfo
	err := r.db.WithContext(ctx).Where("status = ?", 1).Find(&stocks).Error
	return stocks, err
}

func (r *StockRepo) Upsert(ctx context.Context, stock *model.StockBasicInfo) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"symbol", "name", "exchange", "board_code",
			"industry", "is_st", "list_date", "status", "updated_at",
		}),
	}).Create(stock).Error
}

func (r *StockRepo) GetByBoardCode(ctx context.Context, boardCode string) ([]model.StockBasicInfo, error) {
	var stocks []model.StockBasicInfo
	err := r.db.WithContext(ctx).
		Where("board_code = ? AND status = ?", boardCode, 1).
		Find(&stocks).Error
	return stocks, err
}

func (r *StockRepo) Create(ctx context.Context, stock *model.StockBasicInfo) error {
	return r.db.WithContext(ctx).Create(stock).Error
}

func (r *StockRepo) Updates(ctx context.Context, stock *model.StockBasicInfo) error {
	return r.db.WithContext(ctx).Save(stock).Error
}
