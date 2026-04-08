package topic

import (
	"context"
	"fmt"
	"time"

	"stock/dal/dao"
	"stock/dal/repo"
	"stock/model/dal_model"
)

type StockTopicRelationService struct {
	relationRepo  repo.StockTopicRelationRepository
	topicRepo     repo.TopicRepository
	now           func() time.Time
	syncSnapshots func(ctx context.Context, tsCode string) error
}

func NewStockTopicRelationService() *StockTopicRelationService {
	return &StockTopicRelationService{
		relationRepo:  repo.NewStockTopicRelationRepository(),
		topicRepo:     repo.NewTopicRepository(),
		now:           dao.Now,
		syncSnapshots: newSnapshotTopicSyncer().SyncStockSnapshots,
	}
}

func (s *StockTopicRelationService) AddManualRelation(ctx context.Context, tsCode string, topicID int64) ([]dal_model.StockTopicRelation, error) {
	topic, err := s.topicRepo.GetByID(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("get topic by id: %w", err)
	}
	if topic == nil {
		return nil, ErrStockTopicRelationTopicNotFound
	}

	now := s.now()
	relation := &dal_model.StockTopicRelation{
		TsCode:        tsCode,
		TopicID:       topic.ID,
		TopicName:     topic.Name,
		Category:      topic.Category,
		Source:        "manual",
		HitCount:      1,
		FirstSeenDate: &now,
		LastSeenDate:  &now,
		UpdatedAt:     now,
	}
	if err := s.relationRepo.Upsert(ctx, relation); err != nil {
		return nil, fmt.Errorf("upsert stock topic relation: %w", err)
	}
	if err := s.syncSnapshots(ctx, tsCode); err != nil {
		return nil, fmt.Errorf("sync stock snapshots: %w", err)
	}

	relations, err := s.relationRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, fmt.Errorf("get stock topic relations: %w", err)
	}
	return relations, nil
}

func (s *StockTopicRelationService) DeleteRelation(ctx context.Context, tsCode string, topicID int64) ([]dal_model.StockTopicRelation, error) {
	if err := s.relationRepo.Delete(ctx, tsCode, topicID); err != nil {
		return nil, fmt.Errorf("delete stock topic relation: %w", err)
	}
	if err := s.syncSnapshots(ctx, tsCode); err != nil {
		return nil, fmt.Errorf("sync stock snapshots: %w", err)
	}

	relations, err := s.relationRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, fmt.Errorf("get stock topic relations: %w", err)
	}
	return relations, nil
}
