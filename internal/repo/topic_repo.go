package repo

import (
	"context"

	"gorm.io/gorm"
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
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, priority, is_active, created_at, updated_at
		FROM topics WHERE id = ?`, id).Scan(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *TopicRepo) GetByName(ctx context.Context, name string) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, priority, is_active, created_at, updated_at
		FROM topics WHERE name = ?`, name).Scan(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *TopicRepo) Upsert(ctx context.Context, topic *model.Topic) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO topics (name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
		ON CONFLICT (name) DO UPDATE SET
			jiuyan_field_id = COALESCE(EXCLUDED.jiuyan_field_id, topics.jiuyan_field_id),
			first_seen_date = LEAST(topics.first_seen_date, EXCLUDED.first_seen_date),
			last_seen_date = GREATEST(topics.last_seen_date, EXCLUDED.last_seen_date),
			occurrence_count = topics.occurrence_count + EXCLUDED.occurrence_count,
			updated_at = NOW()
	`, topic.Name, topic.Source, topic.JiuyanFieldID, topic.FirstSeenDate, topic.LastSeenDate, topic.OccurrenceCount).Error
}

func (r *TopicRepo) List(ctx context.Context, keyword string, page, pageSize int) ([]model.Topic, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM topics WHERE (? = '' OR name LIKE '%' || ? || '%')`, keyword, keyword).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	var topics []model.Topic
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, priority, is_active, created_at, updated_at
		FROM topics WHERE (? = '' OR name LIKE '%' || ? || '%')
		ORDER BY priority DESC, id ASC LIMIT ? OFFSET ?`, keyword, keyword, pageSize, offset).Scan(&topics).Error
	if err != nil {
		return nil, 0, err
	}
	return topics, total, nil
}

func (r *TopicRepo) GetActiveTopics(ctx context.Context) ([]model.Topic, error) {
	var topics []model.Topic
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, priority, is_active, created_at, updated_at
		FROM topics WHERE is_active = TRUE ORDER BY priority DESC`).Scan(&topics).Error
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func (r *TopicRepo) GetIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.WithContext(ctx).Raw(`SELECT id FROM topics WHERE name = ?`, name).Scan(&id).Error
	return id, err
}

func (r *TopicRepo) Update(ctx context.Context, id int64, name string, isActive bool, priority int) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE topics SET name = ?, is_active = ?, priority = ?, updated_at = NOW()
		WHERE id = ?`, name, isActive, priority, id).Error
}

func (r *TopicRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec(`UPDATE topics SET is_active = FALSE, updated_at = NOW() WHERE id = ?`, id).Error
}

func (r *TopicRepo) Merge(ctx context.Context, sourceID, targetID int64) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE stock_topic_relations SET topic_id = ? WHERE topic_id = ?`, targetID, sourceID).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Exec(`UPDATE topic_synonyms SET topic_id = ? WHERE topic_id = ?`, targetID, sourceID).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec(`UPDATE topics SET is_active = FALSE, updated_at = NOW() WHERE id = ?`, sourceID).Error
}
