package service

import (
	"context"
	"fmt"

	"stock/config"
	"stock/dal/db"
	"stock/internal/external/tushare"
	"stock/internal/pkg/logger"

	"go.uber.org/zap"
)

type ConceptSyncServiceImpl struct {
	tushareClient *tushare.Client
	cfg           *config.TushareConfig
}

var _ ConceptSyncServiceInterface = (*ConceptSyncServiceImpl)(nil)

func NewConceptSyncService(
	tushareClient *tushare.Client,
	cfg *config.TushareConfig,
) *ConceptSyncServiceImpl {
	return &ConceptSyncServiceImpl{
		tushareClient: tushareClient,
		cfg:           cfg,
	}
}

func (s *ConceptSyncServiceImpl) SyncAll(ctx context.Context) error {
	conceptRepo := db.NewConceptRepository()
	conceptDetailRepo := db.NewConceptDetailRepository()

	concepts, err := s.tushareClient.ConceptList(ctx)
	if err != nil {
		return fmt.Errorf("fetch concept list: %w", err)
	}

	for _, c := range concepts {
		if err := conceptRepo.Upsert(ctx, c.Name, c.ID); err != nil {
			logger.Warn("upsert concept failed", zap.String("concept", c.Name), zap.Error(err))
			continue
		}

		details, err := s.tushareClient.ConceptDetail(ctx, c.ID)
		if err != nil {
			logger.Warn("fetch concept detail failed", zap.String("concept_id", c.ID), zap.Error(err))
			continue
		}

		for _, d := range details {
			if err := conceptDetailRepo.Upsert(ctx, d.TsCode, d.ConceptName, d.ID); err != nil {
				logger.Warn("upsert concept detail failed",
					zap.String("ts_code", d.TsCode),
					zap.String("concept", d.ConceptName),
					zap.Error(err))
			}
		}
	}
	return nil
}

func (s *ConceptSyncServiceImpl) SyncIncremental(ctx context.Context) error {
	return s.SyncAll(ctx)
}

func (s *ConceptSyncServiceImpl) BuildMappings(ctx context.Context) error {
	conceptRepo := db.NewConceptRepository()
	topicRepo := db.NewTopicRepository()

	concepts, err := conceptRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("get all concepts: %w", err)
	}

	topics, err := topicRepo.GetActiveTopics(ctx)
	if err != nil {
		return fmt.Errorf("get active topics: %w", err)
	}

	topicMap := make(map[string]int64)
	for _, t := range topics {
		topicMap[t.Name] = t.ID
	}

	for _, c := range concepts {
		if topicID, ok := topicMap[c.ConceptName]; ok {
			if err := conceptRepo.CreateConceptMapping(ctx, c.ConceptName, c.ConceptCode, topicID, "exact"); err != nil {
				logger.Warn("create concept mapping failed",
					zap.String("concept", c.ConceptName),
					zap.Int64("topic_id", topicID),
					zap.Error(err))
			}
		}
	}
	return nil
}

func (s *ConceptSyncServiceImpl) GetUnmappedConcepts(ctx context.Context) ([]string, error) {
	conceptRepo := db.NewConceptRepository()
	return conceptRepo.GetUnmapped(ctx)
}
