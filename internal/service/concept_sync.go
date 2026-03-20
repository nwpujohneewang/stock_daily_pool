package service

import (
	"context"
	"fmt"
	"log"

	"stock/internal/cache"
	"stock/internal/config"
	"stock/internal/external/tushare"
	"stock/internal/repo"
)

type ConceptSyncService struct {
	conceptRepo   *repo.ConceptRepo
	conceptDetail *repo.ConceptDetailRepo
	topicRepo     *repo.TopicRepo
	mappingRepo   *repo.MappingRepo
	tushareClient *tushare.Client
	conceptCache  *cache.ConceptCache
	cfg           *config.TushareConfig
	logger        *log.Logger
}

func NewConceptSyncService(
	conceptRepo *repo.ConceptRepo,
	conceptDetail *repo.ConceptDetailRepo,
	topicRepo *repo.TopicRepo,
	mappingRepo *repo.MappingRepo,
	tushareClient *tushare.Client,
	conceptCache *cache.ConceptCache,
	cfg *config.TushareConfig,
) *ConceptSyncService {
	return &ConceptSyncService{
		conceptRepo:   conceptRepo,
		conceptDetail: conceptDetail,
		topicRepo:     topicRepo,
		mappingRepo:   mappingRepo,
		tushareClient: tushareClient,
		conceptCache:  conceptCache,
		cfg:           cfg,
		logger:        log.Default(),
	}
}

func (s *ConceptSyncService) SyncAll(ctx context.Context) error {
	concepts, err := s.tushareClient.ConceptList(ctx)
	if err != nil {
		return fmt.Errorf("fetch concept list: %w", err)
	}

	for _, c := range concepts {
		if err := s.conceptRepo.Upsert(ctx, c.Name, c.ID); err != nil {
			s.logger.Printf("upsert concept %s failed: %v", c.Name, err)
			continue
		}

		details, err := s.tushareClient.ConceptDetail(ctx, c.ID)
		if err != nil {
			s.logger.Printf("fetch concept detail %s failed: %v", c.ID, err)
			continue
		}

		for _, d := range details {
			if err := s.conceptDetail.Upsert(ctx, d.TsCode, d.ConceptName, d.ID); err != nil {
				s.logger.Printf("upsert concept detail %s -> %s failed: %v", d.TsCode, d.ConceptName, err)
			}
		}
	}
	return nil
}

func (s *ConceptSyncService) SyncIncremental(ctx context.Context) error {
	return s.SyncAll(ctx)
}

func (s *ConceptSyncService) BuildMappings(ctx context.Context) error {
	concepts, err := s.conceptRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("get all concepts: %w", err)
	}

	topics, err := s.topicRepo.GetActiveTopics(ctx)
	if err != nil {
		return fmt.Errorf("get active topics: %w", err)
	}

	topicMap := make(map[string]int64)
	for _, t := range topics {
		topicMap[t.Name] = t.ID
	}

	for _, c := range concepts {
		if topicID, ok := topicMap[c.ConceptName]; ok {
			if err := s.mappingRepo.CreateConceptMapping(ctx, c.ConceptName, c.ConceptCode, topicID, "exact"); err != nil {
				s.logger.Printf("create concept mapping %s -> %d failed: %v", c.ConceptName, topicID, err)
			}
		}
	}
	return nil
}

func (s *ConceptSyncService) GetUnmappedConcepts(ctx context.Context) ([]string, error) {
	return s.conceptRepo.GetUnmapped(ctx)
}
