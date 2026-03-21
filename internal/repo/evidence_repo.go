package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"stock/internal/model"
)

type ClassificationEvidence struct {
	ID               int64           `gorm:"column:id"`
	Date             string          `gorm:"column:date"`
	TsCode           string          `gorm:"column:ts_code"`
	TopicID          *int64          `gorm:"column:topic_id"`
	ClassifyLayer    string          `gorm:"column:classify_layer"`
	Strategy         string          `gorm:"column:strategy"`
	CandidateScores  json.RawMessage `gorm:"column:candidate_scores"`
	EvidenceText     *string         `gorm:"column:evidence_text"`
	Confidence       *float64        `gorm:"column:confidence"`
	CorrectedTopicID *int64          `gorm:"column:corrected_topic_id"`
	CreatedAt        string          `gorm:"column:created_at"`
}

type EvidenceRepo struct {
	db *gorm.DB
}

func NewEvidenceRepo(db *gorm.DB) *EvidenceRepo {
	return &EvidenceRepo{db: db}
}

func (r *EvidenceRepo) Create(ctx context.Context, e model.ClassificationAuditLog) (*model.ClassificationAuditLog, error) {
	err := r.db.WithContext(ctx).Create(&e).Error
	if err != nil {
		return nil, fmt.Errorf("insert evidence: %w", err)
	}
	return &e, nil
}

func (r *EvidenceRepo) GetByStock(ctx context.Context, tsCode string, date string) ([]model.ClassificationAuditLog, error) {
	var results []model.ClassificationAuditLog
	err := r.db.WithContext(ctx).Where("ts_code = ? AND date = ?", tsCode, date).
		Order("created_at DESC").Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}
	return results, nil
}

func (r *EvidenceRepo) GetByID(ctx context.Context, id int64) (*model.ClassificationAuditLog, error) {
	var e model.ClassificationAuditLog
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EvidenceRepo) UpdateCorrectedTopicID(ctx context.Context, id int64, correctedTopicID int64) error {
	return r.db.WithContext(ctx).Model(&model.ClassificationAuditLog{}).
		Where("id = ?", id).Update("corrected_topic_id", correctedTopicID).Error
}

type PaginatedEvidence struct {
	Items     []model.ClassificationAuditLog `json:"items"`
	Total     int64                          `json:"total"`
	Page      int                            `json:"page"`
	PageSize  int                            `json:"page_size"`
	TotalPage int                            `json:"total_page"`
}

func (r *EvidenceRepo) List(ctx context.Context, page, pageSize int) (*PaginatedEvidence, error) {
	offset := (page - 1) * pageSize

	var total int64
	err := r.db.WithContext(ctx).Model(&model.ClassificationAuditLog{}).Count(&total).Error
	if err != nil {
		return nil, fmt.Errorf("count evidence: %w", err)
	}

	var items []model.ClassificationAuditLog
	err = r.db.WithContext(ctx).
		Order("created_at DESC").Limit(pageSize).Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("query evidence list: %w", err)
	}

	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	return &PaginatedEvidence{
		Items:     items,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}, nil
}
