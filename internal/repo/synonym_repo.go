package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type TopicSynonym struct {
	ID        int64  `gorm:"column:id"`
	TopicID   int64  `gorm:"column:topic_id"`
	Synonym   string `gorm:"column:synonym"`
	Source    string `gorm:"column:source"`
	CreatedAt string `gorm:"column:created_at"`
}

type SynonymRepo struct {
	db *gorm.DB
}

func NewSynonymRepo(db *gorm.DB) *SynonymRepo {
	return &SynonymRepo{db: db}
}

func (r *SynonymRepo) GetByTopicID(ctx context.Context, topicID int64) ([]TopicSynonym, error) {
	var results []TopicSynonym
	err := r.db.WithContext(ctx).Where("topic_id = ?", topicID).Order("id").Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query synonyms: %w", err)
	}
	return results, nil
}

func (r *SynonymRepo) GetBySynonym(ctx context.Context, synonym string) (*TopicSynonym, error) {
	var s TopicSynonym
	err := r.db.WithContext(ctx).Where("synonym = ?", synonym).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SynonymRepo) Create(ctx context.Context, synonym TopicSynonym) (*TopicSynonym, error) {
	err := r.db.WithContext(ctx).Create(&synonym).Error
	if err != nil {
		return nil, fmt.Errorf("insert synonym: %w", err)
	}
	return &synonym, nil
}

func (r *SynonymRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&TopicSynonym{}, id).Error
}

func (r *SynonymRepo) BatchCreate(ctx context.Context, synonyms []TopicSynonym) error {
	if len(synonyms) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).CreateInBatches(synonyms, 100).Error
	if err != nil {
		return fmt.Errorf("batch insert synonym: %w", err)
	}
	return nil
}

func (r *SynonymRepo) GetAllSynonymsMap(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.WithContext(ctx).Raw(`SELECT synonym, topic_id FROM topic_synonyms`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query all synonyms: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var synonym string
		var topicID int64
		if err := rows.Scan(&synonym, &topicID); err != nil {
			return nil, fmt.Errorf("scan synonym map: %w", err)
		}
		result[synonym] = topicID
	}
	return result, rows.Err()
}
