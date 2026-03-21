package db

import (
	"context"
	"time"

	"stock/internal/model"
)

var _ AlertRepository = (*AlertRepoImpl)(nil)

type AlertRepository interface {
	Create(ctx context.Context, alert *model.StrategyAlert) (int64, error)
	GetTodayAlerts(ctx context.Context, date string) ([]model.StrategyAlert, error)
	GetHistory(ctx context.Context, startDate, endDate string, topicID *int64, page, pageSize int) ([]model.StrategyAlert, int64, error)
	IsAlerted(ctx context.Context, date, alertKey string) (bool, error)
	MarkAlerted(ctx context.Context, date, alertKey string) error
	GetByDate(ctx context.Context, date string) ([]model.StrategyAlert, error)
}

type AlertRepoImpl struct{}

func NewAlertRepository() *AlertRepoImpl {
	return &AlertRepoImpl{}
}

func (r AlertRepoImpl) Create(ctx context.Context, alert *model.StrategyAlert) (int64, error) {
	err := PostgresStockDB(ctx).Create(alert).Error
	return alert.ID, err
}

func (r AlertRepoImpl) GetTodayAlerts(ctx context.Context, date string) ([]model.StrategyAlert, error) {
	var alerts []model.StrategyAlert
	err := PostgresStockDB(ctx).Where("date = ?", date).Order("trigger_time DESC").Find(&alerts).Error
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r AlertRepoImpl) GetHistory(ctx context.Context, startDate, endDate string, topicID *int64, page, pageSize int) ([]model.StrategyAlert, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	countQuery := PostgresStockDB(ctx).Model(&model.StrategyAlert{}).Where("date >= ? AND date <= ?", startDate, endDate)
	if topicID != nil {
		countQuery = countQuery.Where("topic_id = ?", *topicID)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var alerts []model.StrategyAlert
	query := PostgresStockDB(ctx).Where("date >= ? AND date <= ?", startDate, endDate).
		Order("trigger_time DESC").Limit(pageSize).Offset(offset)
	if topicID != nil {
		query = query.Where("topic_id = ?", *topicID)
	}
	err := query.Find(&alerts).Error
	return alerts, total, err
}

func (r AlertRepoImpl) IsAlerted(ctx context.Context, date, alertKey string) (bool, error) {
	var exists bool
	err := PostgresStockDB(ctx).Raw(
		`SELECT EXISTS(SELECT 1 FROM strategy_alerts WHERE date = ? AND ts_code || ':'::text || COALESCE(topic_id::text, '') = ?)`,
		date, alertKey).Scan(&exists).Error
	return exists, err
}

func (r AlertRepoImpl) MarkAlerted(ctx context.Context, date, alertKey string) error {
	var tsCode string
	var topicID *int64

	for i := len(alertKey) - 1; i >= 0; i-- {
		if alertKey[i] == ':' {
			tsCode = alertKey[:i]
			if i+1 < len(alertKey) {
				var tid int64
				for _, c := range alertKey[i+1:] {
					if c >= '0' && c <= '9' {
						tid = tid*10 + int64(c-'0')
					}
				}
				topicID = &tid
			}
			break
		}
	}
	if tsCode == "" {
		tsCode = alertKey
	}

	dateParsed, _ := time.Parse("2006-01-02", date)
	return PostgresStockDB(ctx).Create(&model.StrategyAlert{
		Date:      dateParsed,
		TsCode:    tsCode,
		TopicID:   topicID,
		AlertType: 0,
		Notified:  false,
	}).Error
}

func (r AlertRepoImpl) GetByDate(ctx context.Context, date string) ([]model.StrategyAlert, error) {
	return r.GetTodayAlerts(ctx, date)
}
