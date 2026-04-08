package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

var _ TopicDAO = (*topicDAOImpl)(nil)

type TopicDAO interface {
	GetByID(ctx context.Context, id int64) (*dal_model.Topic, error)
	GetByIDs(ctx context.Context, ids []int64) ([]dal_model.Topic, error)
	GetByName(ctx context.Context, name string) (*dal_model.Topic, error)
	GetByNames(ctx context.Context, names []string) ([]dal_model.Topic, error)
	GetNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
	Upsert(ctx context.Context, topic *dal_model.Topic) error
	UpsertBatch(ctx context.Context, topics []dal_model.Topic) error
	List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.Topic, int64, error)
	GetActiveTopics(ctx context.Context) ([]dal_model.Topic, error)
	GetIDByName(ctx context.Context, name string) (int64, error)
	Update(ctx context.Context, id int64, name string, isActive bool, priority int) error
	Delete(ctx context.Context, id int64) error
	Merge(ctx context.Context, sourceID, targetID int64) error
}

type topicDAOImpl struct{}

func NewTopicDAO() TopicDAO {
	return &topicDAOImpl{}
}

func (r topicDAOImpl) GetByID(ctx context.Context, id int64) (*dal_model.Topic, error) {
	var topic dal_model.Topic
	err := PostgresStockDB(ctx).Where("id = ?", id).First(&topic).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &topic, nil
}

func (r topicDAOImpl) GetByIDs(ctx context.Context, ids []int64) ([]dal_model.Topic, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var topics []dal_model.Topic
	err := PostgresStockDB(ctx).Where("id IN ?", ids).Find(&topics).Error
	return topics, err
}

func (r topicDAOImpl) GetByName(ctx context.Context, name string) (*dal_model.Topic, error) {
	var topic dal_model.Topic
	err := PostgresStockDB(ctx).Where("name = ?", name).First(&topic).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &topic, nil
}

func (r topicDAOImpl) GetByNames(ctx context.Context, names []string) ([]dal_model.Topic, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var topics []dal_model.Topic
	err := PostgresStockDB(ctx).Where("name IN ?", names).Find(&topics).Error
	return topics, err
}

func (r topicDAOImpl) GetNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return make(map[int64]string), nil
	}
	var topics []dal_model.Topic
	err := PostgresStockDB(ctx).Select("id, name").Where("id IN ?", ids).Find(&topics).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(topics))
	for _, t := range topics {
		result[t.ID] = t.Name
	}
	return result, nil
}

func (r topicDAOImpl) Upsert(ctx context.Context, topic *dal_model.Topic) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"category", "jiuyan_field_id", "first_seen_date", "last_seen_date",
			"occurrence_count", "updated_at",
		}),
	}).Create(topic).Error
}

func (r topicDAOImpl) UpsertBatch(ctx context.Context, topics []dal_model.Topic) error {
	if len(topics) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"category", "jiuyan_field_id", "first_seen_date", "last_seen_date",
			"occurrence_count", "updated_at",
		}),
	}).CreateInBatches(&topics, 500).Error
}

func (r topicDAOImpl) List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.Topic, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	query := PostgresStockDB(ctx).Model(&dal_model.Topic{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var topics []dal_model.Topic
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

func (r topicDAOImpl) GetActiveTopics(ctx context.Context) ([]dal_model.Topic, error) {
	var topics []dal_model.Topic
	err := PostgresStockDB(ctx).Where("is_active = ?", true).Order("priority DESC").Find(&topics).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return topics, nil
}

func (r topicDAOImpl) GetIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := PostgresStockDB(ctx).Model(&dal_model.Topic{}).Select("id").Where("name = ?", name).Scan(&id).Error
	return id, err
}

func (r topicDAOImpl) Update(ctx context.Context, id int64, name string, isActive bool, priority int) error {
	return PostgresStockDB(ctx).Model(&dal_model.Topic{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":       name,
		"is_active":  isActive,
		"priority":   priority,
		"updated_at": Now(),
	}).Error
}

func (r topicDAOImpl) Delete(ctx context.Context, id int64) error {
	return PostgresStockDB(ctx).Model(&dal_model.Topic{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r topicDAOImpl) Merge(ctx context.Context, sourceID, targetID int64) error {
	return PostgresStockDB(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: Update stock_topic_relations
		if err := tx.Model(&dal_model.StockTopicRelation{}).Where("topic_id = ?", sourceID).Update("topic_id", targetID).Error; err != nil {
			return err
		}
		// Step 2: Update topic_synonyms
		if err := tx.Model(&dal_model.TopicSynonym{}).Where("topic_id = ?", sourceID).Update("topic_id", targetID).Error; err != nil {
			return err
		}
		// Step 3: Deactivate source topic
		return tx.Model(&dal_model.Topic{}).Where("id = ?", sourceID).Update("is_active", false).Error
	})
}
