package repo

import (
	"context"

	"gorm.io/gorm"
	"stock/internal/model"
)

type AlertRepo struct {
	db *gorm.DB
}

func NewAlertRepo(db *gorm.DB) *AlertRepo {
	return &AlertRepo{db: db}
}

func (r *AlertRepo) Create(ctx context.Context, alert *model.StrategyAlert) (int64, error) {
	var id int64
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO strategy_alerts (date, ts_code, stock_name, topic_id, topic_name, alert_type, trigger_price, trigger_time, prev_day_pct, extra_info, notified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		alert.Date, alert.TsCode, alert.StockName, alert.TopicID, alert.TopicName,
		alert.AlertType, alert.TriggerPrice, alert.TriggerTime, alert.PrevDayPct, alert.ExtraInfo, alert.Notified).Scan(&id).Error
	return id, err
}

func (r *AlertRepo) GetTodayAlerts(ctx context.Context, date string) ([]model.StrategyAlert, error) {
	var alerts []model.StrategyAlert
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, date, ts_code, stock_name, topic_id, topic_name, alert_type, trigger_price, trigger_time, prev_day_pct, extra_info, notified, created_at
		FROM strategy_alerts WHERE date = ? ORDER BY trigger_time DESC`, date).Scan(&alerts).Error
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r *AlertRepo) GetHistory(ctx context.Context, startDate, endDate string, topicID *int64, page, pageSize int) ([]model.StrategyAlert, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	countQuery := `SELECT COUNT(*) FROM strategy_alerts WHERE date >= ? AND date <= ?`
	args := []interface{}{startDate, endDate}
	if topicID != nil {
		countQuery += ` AND topic_id = ?`
		args = append(args, *topicID)
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	query := `SELECT id, date, ts_code, stock_name, topic_id, topic_name, alert_type, trigger_price, trigger_time, prev_day_pct, extra_info, notified, created_at
		FROM strategy_alerts WHERE date >= ? AND date <= ?`
	if topicID != nil {
		query += ` AND topic_id = ?`
		query += ` ORDER BY trigger_time DESC LIMIT ? OFFSET ?`
		var alerts []model.StrategyAlert
		err := r.db.WithContext(ctx).Raw(query, startDate, endDate, *topicID, pageSize, offset).Scan(&alerts).Error
		return alerts, total, err
	}
	query += ` ORDER BY trigger_time DESC LIMIT ? OFFSET ?`
	var alerts []model.StrategyAlert
	err := r.db.WithContext(ctx).Raw(query, startDate, endDate, pageSize, offset).Scan(&alerts).Error
	return alerts, total, err
}

func (r *AlertRepo) IsAlerted(ctx context.Context, date, alertKey string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS(SELECT 1 FROM strategy_alerts WHERE date = ? AND ts_code || ':'::text || COALESCE(topic_id::text, '') = ?)`,
		date, alertKey).Scan(&exists).Error
	return exists, err
}

func (r *AlertRepo) MarkAlerted(ctx context.Context, date, alertKey string) error {
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

	return r.db.WithContext(ctx).Exec(`
		INSERT INTO strategy_alerts (date, ts_code, topic_id, alert_type, notified)
		VALUES (?, ?, ?, 0, FALSE)`,
		date, tsCode, topicID).Error
}

func (r *AlertRepo) GetByDate(ctx context.Context, date string) ([]model.StrategyAlert, error) {
	return r.GetTodayAlerts(ctx, date)
}
