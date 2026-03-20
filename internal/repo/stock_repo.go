package repo

import (
	"context"

	"gorm.io/gorm"
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
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, symbol, name, exchange, board_code, industry, is_st, list_date, status, created_at, updated_at
		FROM stock_basic_info WHERE ts_code = ?`, tsCode).Scan(&stock).Error
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (r *StockRepo) GetActiveStocks(ctx context.Context) ([]model.StockBasicInfo, error) {
	var stocks []model.StockBasicInfo
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, symbol, name, exchange, board_code, industry, is_st, list_date, status, created_at, updated_at
		FROM stock_basic_info WHERE status = 1`).Scan(&stocks).Error
	if err != nil {
		return nil, err
	}
	return stocks, nil
}

func (r *StockRepo) Upsert(ctx context.Context, stock *model.StockBasicInfo) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO stock_basic_info (ts_code, symbol, name, exchange, board_code, industry, is_st, list_date, status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON CONFLICT (ts_code) DO UPDATE SET
			name = EXCLUDED.name,
			exchange = EXCLUDED.exchange,
			board_code = EXCLUDED.board_code,
			industry = EXCLUDED.industry,
			is_st = EXCLUDED.is_st,
			list_date = EXCLUDED.list_date,
			status = EXCLUDED.status,
			updated_at = NOW()
	`, stock.TsCode, stock.Symbol, stock.Name, stock.Exchange, stock.BoardCode, stock.Industry, stock.IsST, stock.ListDate, stock.Status).Error
}

func (r *StockRepo) GetByBoardCode(ctx context.Context, boardCode string) ([]model.StockBasicInfo, error) {
	var stocks []model.StockBasicInfo
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, symbol, name, exchange, board_code, industry, is_st, list_date, status, created_at, updated_at
		FROM stock_basic_info WHERE board_code = ? AND status = 1`, boardCode).Scan(&stocks).Error
	if err != nil {
		return nil, err
	}
	return stocks, nil
}
