package repo

import (
	"context"

	"gorm.io/gorm"
)

type Concept struct {
	ID          int64  `gorm:"column:id"`
	ConceptCode string `gorm:"column:concept_code"`
	ConceptName string `gorm:"column:concept_name"`
	Source      string `gorm:"column:source"`
}

type ConceptRepo struct {
	db *gorm.DB
}

func NewConceptRepo(db *gorm.DB) *ConceptRepo {
	return &ConceptRepo{db: db}
}

func (r *ConceptRepo) Upsert(ctx context.Context, conceptName, conceptCode string) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO tushare_concepts (concept_code, concept_name, source, updated_at)
		VALUES (?, ?, 'tushare', NOW())
		ON CONFLICT (concept_code) DO UPDATE SET
			concept_name = EXCLUDED.concept_name,
			updated_at = NOW()
	`, conceptCode, conceptName).Error
}

func (r *ConceptRepo) GetAll(ctx context.Context) ([]Concept, error) {
	var concepts []Concept
	err := r.db.WithContext(ctx).Raw(`SELECT id, concept_code, concept_name, source FROM tushare_concepts`).Scan(&concepts).Error
	if err != nil {
		return nil, err
	}
	return concepts, nil
}

func (r *ConceptRepo) GetByCode(ctx context.Context, conceptCode string) (*Concept, error) {
	var c Concept
	err := r.db.WithContext(ctx).Raw(`SELECT id, concept_code, concept_name, source FROM tushare_concepts WHERE concept_code = ?`, conceptCode).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConceptRepo) List(ctx context.Context, keyword string, page, pageSize int) ([]Concept, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM tushare_concepts WHERE (? = '' OR concept_name LIKE '%' || ? || '%')`, keyword, keyword).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	var concepts []Concept
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, concept_code, concept_name, source FROM tushare_concepts
		WHERE (? = '' OR concept_name LIKE '%' || ? || '%')
		ORDER BY concept_name LIMIT ? OFFSET ?`, keyword, keyword, pageSize, offset).Scan(&concepts).Error
	if err != nil {
		return nil, 0, err
	}
	return concepts, total, nil
}

func (r *ConceptRepo) GetUnmapped(ctx context.Context) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.concept_name FROM tushare_concepts c
		LEFT JOIN topic_concepts tc ON c.concept_name = tc.concept_name
		WHERE tc.id IS NULL`).Scan(&names).Error
	if err != nil {
		return nil, err
	}
	return names, nil
}
