package repo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"stock/internal/model"
)

type TopicRepo struct {
	db *gorm.DB
}

func NewTopicRepo(db *gorm.DB) *TopicRepo {
	return &TopicRepo{db: db}
}

func (r *TopicRepo) GetByID(ctx context.Context, id int64) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *TopicRepo) GetByName(ctx context.Context, name string) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *TopicRepo) Upsert(ctx context.Context, topic *model.Topic) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"jiuyan_field_id", "first_seen_date", "last_seen_date",
			"occurrence_count", "updated_at",
		}),
	}).Create(topic).Error
}

func (r *TopicRepo) List(ctx context.Context, keyword string, page, pageSize int) ([]model.Topic, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	query := r.db.WithContext(ctx).Model(&model.Topic{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var topics []model.Topic
	err := r.db.WithContext(ctx).
		Where("? = '' OR name LIKE ?", keyword, "%"+keyword+"%").
		Order("priority DESC, id ASC").
		Limit(pageSize).Offset(offset).
		Find(&topics).Error
	if err != nil {
		return nil, 0, err
	}
	return topics, total, nil
}

func (r *TopicRepo) GetActiveTopics(ctx context.Context) ([]model.Topic, error) {
	var topics []model.Topic
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("priority DESC").Find(&topics).Error
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func (r *TopicRepo) GetIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.WithContext(ctx).Model(&model.Topic{}).Select("id").Where("name = ?", name).Scan(&id).Error
	return id, err
}

func (r *TopicRepo) Update(ctx context.Context, id int64, name string, isActive bool, priority int) error {
	return r.db.WithContext(ctx).Model(&model.Topic{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":       name,
		"is_active":  isActive,
		"priority":   priority,
		"updated_at": time.Now(),
	}).Error
}

func (r *TopicRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.Topic{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r *TopicRepo) Merge(ctx context.Context, sourceID, targetID int64) error {
	if err := r.db.WithContext(ctx).Model(&model.StockTopicRelation{}).Where("topic_id = ?", sourceID).Update("topic_id", targetID).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&TopicSynonym{}).Where("topic_id = ?", sourceID).Update("topic_id", targetID).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&model.Topic{}).Where("id = ?", sourceID).Update("is_active", false).Error
}
