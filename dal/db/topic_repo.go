package db

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
	"stock/internal/model"
)

var _ TopicRepository = (*TopicRepoImpl)(nil)

type TopicRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Topic, error)
	GetByName(ctx context.Context, name string) (*model.Topic, error)
	Upsert(ctx context.Context, topic *model.Topic) error
	List(ctx context.Context, keyword string, page, pageSize int) ([]model.Topic, int64, error)
	GetActiveTopics(ctx context.Context) ([]model.Topic, error)
	GetIDByName(ctx context.Context, name string) (int64, error)
	Update(ctx context.Context, id int64, name string, isActive bool, priority int) error
	Delete(ctx context.Context, id int64) error
	Merge(ctx context.Context, sourceID, targetID int64) error
}

type TopicRepoImpl struct{}

func NewTopicRepository() *TopicRepoImpl {
	return &TopicRepoImpl{}
}

func (r TopicRepoImpl) GetByID(ctx context.Context, id int64) (*model.Topic, error) {
	var topic model.Topic
	err := PostgresStockDB(ctx).Where("id = ?", id).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r TopicRepoImpl) GetByName(ctx context.Context, name string) (*model.Topic, error) {
	var topic model.Topic
	err := PostgresStockDB(ctx).Where("name = ?", name).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r TopicRepoImpl) Upsert(ctx context.Context, topic *model.Topic) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"jiuyan_field_id", "first_seen_date", "last_seen_date",
			"occurrence_count", "updated_at",
		}),
	}).Create(topic).Error
}

func (r TopicRepoImpl) List(ctx context.Context, keyword string, page, pageSize int) ([]model.Topic, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	query := PostgresStockDB(ctx).Model(&model.Topic{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var topics []model.Topic
	err := PostgresStockDB(ctx).
		Where("? = '' OR name LIKE ?", keyword, "%"+keyword+"%").
		Order("priority DESC, id ASC").
		Limit(pageSize).Offset(offset).
		Find(&topics).Error
	if err != nil {
		return nil, 0, err
	}
	return topics, total, nil
}

func (r TopicRepoImpl) GetActiveTopics(ctx context.Context) ([]model.Topic, error) {
	var topics []model.Topic
	err := PostgresStockDB(ctx).Where("is_active = ?", true).Order("priority DESC").Find(&topics).Error
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func (r TopicRepoImpl) GetIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := PostgresStockDB(ctx).Model(&model.Topic{}).Select("id").Where("name = ?", name).Scan(&id).Error
	return id, err
}

func (r TopicRepoImpl) Update(ctx context.Context, id int64, name string, isActive bool, priority int) error {
	return PostgresStockDB(ctx).Model(&model.Topic{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":       name,
		"is_active":  isActive,
		"priority":   priority,
		"updated_at": time.Now(),
	}).Error
}

func (r TopicRepoImpl) Delete(ctx context.Context, id int64) error {
	return PostgresStockDB(ctx).Model(&model.Topic{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r TopicRepoImpl) Merge(ctx context.Context, sourceID, targetID int64) error {
	if err := PostgresStockDB(ctx).Model(&model.StockTopicRelation{}).Where("topic_id = ?", sourceID).Update("topic_id", targetID).Error; err != nil {
		return err
	}
	if err := PostgresStockDB(ctx).Model(&model.TopicSynonym{}).Where("topic_id = ?", sourceID).Update("topic_id", targetID).Error; err != nil {
		return err
	}
	return PostgresStockDB(ctx).Model(&model.Topic{}).Where("id = ?", sourceID).Update("is_active", false).Error
}
