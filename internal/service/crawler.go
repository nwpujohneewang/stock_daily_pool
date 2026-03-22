package service

import (
	"context"
	"fmt"
	"stock/model/dal_model"
	"time"

	"stock/config"
	"stock/dal/db"
	"stock/internal/external/jiuyan"
	"stock/internal/pkg/converter"
	"stock/internal/pkg/logger"

	"go.uber.org/zap"
)

type CrawlerServiceImpl struct {
	cfg *config.JiuyanConfig
}

var _ CrawlerServiceInterface = (*CrawlerServiceImpl)(nil)

func NewCrawlerService(cfg *config.JiuyanConfig) *CrawlerServiceImpl {
	return &CrawlerServiceImpl{
		cfg: cfg,
	}
}

func (s *CrawlerServiceImpl) CrawlDate(ctx context.Context, date string) error {
	data, err := jiuyan.FetchFieldData(ctx, date)
	if err != nil {
		return fmt.Errorf("fetch jiuyan data: %w", err)
	}

	topicRepo := db.NewTopicRepository()
	mappingRepo := db.NewStockTopicRelationRepository()
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
			logger.Warn("upsert topic failed", zap.String("topic", topicName), zap.Error(err))
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
				logger.Warn("upsert mapping failed",
					zap.String("ts_code", tsCode),
					zap.Int64("topic_id", topicID),
					zap.Error(err))
			}
		}
	}
	return nil
}

func (s *CrawlerServiceImpl) CrawlHistory(ctx context.Context, startDate string) error {
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
			logger.Warn("crawl date failed", zap.String("date", dateStr), zap.Error(err))
		}
	}
	return nil
}

func (s *CrawlerServiceImpl) CrawlToday(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1)
	dateStr := yesterday.Format("2006-01-02")
	return s.CrawlDate(ctx, dateStr)
}
