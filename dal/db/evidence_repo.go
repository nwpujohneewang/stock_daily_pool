package db

import (
	"context"
	"encoding/json"
	"fmt"

	"stock/internal/model"
)

var _ EvidenceRepository = (*EvidenceRepoImpl)(nil)

type EvidenceRepository interface {
	Create(ctx context.Context, e model.ClassificationAuditLog) (*model.ClassificationAuditLog, error)
	GetByStock(ctx context.Context, tsCode string, date string) ([]model.ClassificationAuditLog, error)
	GetByID(ctx context.Context, id int64) (*model.ClassificationAuditLog, error)
	UpdateCorrectedTopicID(ctx context.Context, id int64, correctedTopicID int64) error
	List(ctx context.Context, page, pageSize int) (*PaginatedEvidence, error)
}

type EvidenceRepoImpl struct{}

func NewEvidenceRepository() *EvidenceRepoImpl {
	return &EvidenceRepoImpl{}
}

func (r EvidenceRepoImpl) Create(ctx context.Context, e model.ClassificationAuditLog) (*model.ClassificationAuditLog, error) {
	err := PostgresStockDB(ctx).Create(&e).Error
	if err != nil {
		return nil, fmt.Errorf("insert evidence: %w", err)
	}
	return &e, nil
}

func (r EvidenceRepoImpl) GetByStock(ctx context.Context, tsCode string, date string) ([]model.ClassificationAuditLog, error) {
	var results []model.ClassificationAuditLog
	err := PostgresStockDB(ctx).Where("ts_code = ? AND date = ?", tsCode, date).
		Order("created_at DESC").Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}
	return results, nil
}

func (r EvidenceRepoImpl) GetByID(ctx context.Context, id int64) (*model.ClassificationAuditLog, error) {
	var e model.ClassificationAuditLog
	err := PostgresStockDB(ctx).Where("id = ?", id).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r EvidenceRepoImpl) UpdateCorrectedTopicID(ctx context.Context, id int64, correctedTopicID int64) error {
	return PostgresStockDB(ctx).Model(&model.ClassificationAuditLog{}).
		Where("id = ?", id).Update("corrected_topic_id", correctedTopicID).Error
}

type PaginatedEvidence struct {
	Items     []model.ClassificationAuditLog `json:"items"`
	Total     int64                          `json:"total"`
	Page      int                            `json:"page"`
	PageSize  int                            `json:"page_size"`
	TotalPage int                            `json:"total_page"`
}

func (r EvidenceRepoImpl) List(ctx context.Context, page, pageSize int) (*PaginatedEvidence, error) {
	offset := (page - 1) * pageSize

	var total int64
	err := PostgresStockDB(ctx).Model(&model.ClassificationAuditLog{}).Count(&total).Error
	if err != nil {
		return nil, fmt.Errorf("count evidence: %w", err)
	}

	var items []model.ClassificationAuditLog
	err = PostgresStockDB(ctx).
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

// ---- json.RawMessage alias for compatibility ----
type rawMessage = json.RawMessage
