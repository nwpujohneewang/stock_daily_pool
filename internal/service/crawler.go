package service

import (
	"context"
	"fmt"
	"log"
	"stock/model/dal_model"
	"time"

	"stock/config"
	"stock/dal/db"
	"stock/internal/external/jiuyan"
	"stock/internal/pkg/converter"
)

type CrawlerService struct {
	cfg    *config.JiuyanConfig
	logger *log.Logger
}

func NewCrawlerService(cfg *config.JiuyanConfig) *CrawlerService {
	return &CrawlerService{
		cfg:    cfg,
		logger: log.Default(),
	}
}

func (s *CrawlerService) CrawlDate(ctx context.Context, date string) error {
	data, err := jiuyan.FetchFieldData(ctx, date)
	if err != nil {
		return fmt.Errorf("fetch jiuyan data: %w", err)
	}

	topicRepo := db.NewTopicRepository()
	mappingRepo := db.NewMappingRepository()
	synonymRepo := db.NewSynonymRepository()

	for _, field := range data {
		topicName := field.Name

		normalized, err := synonymRepo.GetBySynonym(ctx, topicName)
		if err == nil && normalized != nil {
			topicName = normalized.Synonym
		}

		now := time.Now()
		topic := &dal_model.Topic{
			Name:          topicName,
			Source:        "jiuyan",
			JiuyanFieldID: &field.ActionFieldID,
			LastSeenDate:  &now,
			FirstSeenDate: &now,
		}
		if err := topicRepo.Upsert(ctx, topic); err != nil {
			s.logger.Printf("upsert topic %s failed: %v", topicName, err)
			continue
		}

		topicID, err := topicRepo.GetIDByName(ctx, topicName)
		if err != nil {
			continue
		}

		for _, stock := range field.List {
			tsCode := converter.JiuyanToTushare(stock.Code)
			if tsCode == "" {
				continue
			}

			mapping := &dal_model.StockTopicRelation{
				TsCode:        tsCode,
				TopicID:       topicID,
				Source:        "jiuyan",
				HitCount:      1,
				LastSeenDate:  &now,
				FirstSeenDate: &now,
			}
			if err := mappingRepo.Upsert(ctx, mapping); err != nil {
				s.logger.Printf("upsert mapping %s -> %d failed: %v", tsCode, topicID, err)
			}
		}
	}
	return nil
}

func (s *CrawlerService) CrawlHistory(ctx context.Context, startDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("parse start date: %w", err)
	}

	for d := start; d.Before(time.Now()); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		dateStr := d.Format("2006-01-02")
		if err := s.CrawlDate(ctx, dateStr); err != nil {
			s.logger.Printf("crawl date %s failed: %v", dateStr, err)
		}
	}
	return nil
}

func (s *CrawlerService) CrawlToday(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1)
	dateStr := yesterday.Format("2006-01-02")
	return s.CrawlDate(ctx, dateStr)
}
