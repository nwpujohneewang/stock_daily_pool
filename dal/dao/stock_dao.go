package dao

import (
	"context"
	"errors"
	"stock/external/tushare"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

var _ StockDAO = (*stockDAOImpl)(nil)

type StockDAO interface {
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

type stockDAOImpl struct{}

func NewStockDAO() StockDAO {
	return &stockDAOImpl{}
}

func (r stockDAOImpl) Search(ctx context.Context, query string, limit int) ([]dal_model.StockBasicInfo, error) {
	var stocks []dal_model.StockBasicInfo
	db := PostgresStockDB(ctx)
	if query != "" {
		db = db.Where("ts_code LIKE ? OR symbol LIKE ? OR name LIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%")
	}
	err := db.Limit(limit).Find(&stocks).Error
	return stocks, err
}

func (r stockDAOImpl) GetAllStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	var stocks []dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).Find(&stocks).Error
	return stocks, err
}

func (r stockDAOImpl) GetPaginatedStocks(ctx context.Context, query, topic, category string, page, pageSize int) ([]dal_model.StockBasicInfo, int64, error) {
	var stocks []dal_model.StockBasicInfo
	var total int64
	db := PostgresStockDB(ctx).Model(&dal_model.StockBasicInfo{})

	if query != "" {
		db = db.Where("ts_code LIKE ? OR symbol LIKE ? OR name LIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%")
	}

	if topic != "" || category != "" {
		// Join with stock_topic_relations using subquery for efficiency
		subQuery := PostgresStockDB(ctx).Table("stock_topic_relations").Select("DISTINCT ts_code")
		if topic != "" {
			subQuery = subQuery.Where("topic_name LIKE ?", "%"+topic+"%")
		}
		if category != "" {
			subQuery = subQuery.Where("category = ?", category)
		}
		db = db.Where("ts_code IN (?)", subQuery)
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&stocks).Error
	return stocks, total, err
}

func (r stockDAOImpl) GetByTsCode(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error) {
	var stock dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &stock, nil
}

func (r stockDAOImpl) GetByTsCodes(ctx context.Context, tsCodes []string) ([]dal_model.StockBasicInfo, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}
	var stocks []dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("ts_code IN ?", tsCodes).Find(&stocks).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return stocks, nil
}

func (r stockDAOImpl) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	var stocks []dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).Where("status = ?", 1).Find(&stocks).Error
	return stocks, err
}

func (r stockDAOImpl) Upsert(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"symbol", "name", "exchange", "board_code",
			"industry", "is_st", "list_date", "status",
			"total_mv", "circ_mv", "updated_at",
		}),
	}).Create(stock).Error
}

func (r stockDAOImpl) UpsertBatch(ctx context.Context, stocks []dal_model.StockBasicInfo) error {
	if len(stocks) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"symbol", "name", "exchange", "board_code",
			"industry", "is_st", "list_date", "status",
			"total_mv", "circ_mv", "updated_at",
		}),
	}).CreateInBatches(&stocks, 500).Error
}

func (r stockDAOImpl) SetSTBatch(ctx context.Context, tsCodes []string, isST bool) error {
	if len(tsCodes) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).
		Model(&dal_model.StockBasicInfo{}).
		Where("ts_code IN ?", tsCodes).
		Update("is_st", isST).Error
}

func (r stockDAOImpl) GetByBoardCode(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error) {
	var stocks []dal_model.StockBasicInfo
	err := PostgresStockDB(ctx).
		Where("board_code = ? AND status = ?", boardCode, 1).
		Find(&stocks).Error
	return stocks, err
}

func (r stockDAOImpl) Create(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Create(stock).Error
}

func (r stockDAOImpl) Updates(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	return PostgresStockDB(ctx).Save(stock).Error
}

func (r stockDAOImpl) UpdateMarketValueBatch(ctx context.Context, items []tushare.DailyBasicItem) error {
	if len(items) == 0 {
		return nil
	}
	for i := 0; i < len(items); i += 500 {
		end := i + 500
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		for _, item := range batch {
			PostgresStockDB(ctx).
				Model(&dal_model.StockBasicInfo{}).
				Where("ts_code = ?", item.TsCode).
				Updates(map[string]interface{}{
					"total_mv": item.TotalMv,
					"circ_mv":  item.CircMv,
				})
		}
	}
	return nil
}
