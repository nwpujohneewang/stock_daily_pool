package topic

import (
	"context"
	"time"

	"stock/dal/repo"
	"stock/internal/service/classify"
	"stock/model/dal_model"
)

type snapshotTopicSyncer struct {
	snapshotRepo repo.SnapshotRepository
	classifyFn   func(ctx context.Context, tsCode, date string, quoteTime time.Time) ([]dal_model.TopicRelation, error)
}

func newSnapshotTopicSyncer() *snapshotTopicSyncer {
	svc := classify.NewClassifyService()
	return &snapshotTopicSyncer{
		snapshotRepo: repo.NewSnapshotRepository(),
		classifyFn: func(ctx context.Context, tsCode, date string, quoteTime time.Time) ([]dal_model.TopicRelation, error) {
			return svc.ClassifyStock(ctx, tsCode, date, quoteTime)
		},
	}
}

func (s *snapshotTopicSyncer) SyncStockSnapshots(ctx context.Context, tsCode string) error {
	snapshots, err := s.snapshotRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		date := snapshot.Date.Format("2006-01-02")
		quoteTime := time.Date(snapshot.Date.Year(), snapshot.Date.Month(), snapshot.Date.Day(), 15, 0, 0, 0, snapshot.Date.Location())
		relations, err := s.classifyFn(ctx, tsCode, date, quoteTime)
		if err != nil {
			return err
		}

		var topicID *int64
		if len(relations) > 0 {
			id := relations[0].TopicID
			topicID = &id
		}
		if err := s.snapshotRepo.UpdateTopicIDByTsCodeAndDate(ctx, date, tsCode, topicID); err != nil {
			return err
		}
	}
	return nil
}
