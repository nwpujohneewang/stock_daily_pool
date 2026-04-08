package dao

import (
	"context"

	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

type SnapshotDAO interface {
	DeleteByDate(ctx context.Context, date string) error
	DeleteBeforeOrEqualDate(ctx context.Context, cutoffDate string) error
	InsertBatch(ctx context.Context, snapshots []dal_model.DailyStockSnapshot) error
	GetByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error)
	GetLimitUpByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error)
	GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.DailyStockSnapshot, error)
	UpdateTopicIDByTsCodeAndDate(ctx context.Context, date, tsCode string, topicID *int64) error
}

type snapshotDAOImpl struct{}

func NewSnapshotDAO() SnapshotDAO {
	return &snapshotDAOImpl{}
}

func (r snapshotDAOImpl) DeleteByDate(ctx context.Context, date string) error {
	if err := validateDateFormat(date); err != nil {
		return err
	}
	return PostgresStockDB(ctx).
		Where("date = ?", date).
		Delete(&dal_model.DailyStockSnapshot{}).Error
}

func (r snapshotDAOImpl) DeleteBeforeOrEqualDate(ctx context.Context, cutoffDate string) error {
	if err := validateDateFormat(cutoffDate); err != nil {
		return err
	}
	return PostgresStockDB(ctx).
		Where("date <= ?", cutoffDate).
		Delete(&dal_model.DailyStockSnapshot{}).Error
}

func (r snapshotDAOImpl) InsertBatch(ctx context.Context, snapshots []dal_model.DailyStockSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "ts_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "change_pct", "is_limit_up", "limit_times",
			"consecutive_strong_days", "total_mv", "vol", "amount",
		}),
	}).CreateInBatches(&snapshots, 500).Error
}

func (r snapshotDAOImpl) GetByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error) {
	if err := validateDateFormat(date); err != nil {
		return nil, err
	}
	var snapshots []dal_model.DailyStockSnapshot
	err := PostgresStockDB(ctx).Where("date = ?", date).Find(&snapshots).Error
	return snapshots, err
}

func (r snapshotDAOImpl) GetLimitUpByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error) {
	if err := validateDateFormat(date); err != nil {
		return nil, err
	}
	var snapshots []dal_model.DailyStockSnapshot
	err := PostgresStockDB(ctx).
		Where("date = ? AND is_limit_up = true", date).
		Find(&snapshots).Error
	return snapshots, err
}

func (r snapshotDAOImpl) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.DailyStockSnapshot, error) {
	var snapshots []dal_model.DailyStockSnapshot
	err := PostgresStockDB(ctx).
		Where("ts_code = ?", tsCode).
		Order("date ASC").
		Find(&snapshots).Error
	return snapshots, err
}

func (r snapshotDAOImpl) UpdateTopicIDByTsCodeAndDate(ctx context.Context, date, tsCode string, topicID *int64) error {
	if err := validateDateFormat(date); err != nil {
		return err
	}
	return PostgresStockDB(ctx).
		Model(&dal_model.DailyStockSnapshot{}).
		Where("date = ? AND ts_code = ?", date, tsCode).
		Update("topic_id", topicID).Error
}
