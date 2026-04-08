// internal/service/focus_service.go
package focus

import (
	"context"
	"fmt"

	"stock/dal/cache"
	"stock/dal/repo"
)

type FocusServiceImpl struct{}

func NewFocusService() *FocusServiceImpl {
	return &FocusServiceImpl{}
}

func (s *FocusServiceImpl) Get(ctx context.Context, date string) ([]FocusTopicResult, error) {
	focusCache := cache.NewFocusCache()
	topicIDs, err := focusCache.GetFocusTopics(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("get focus topics from cache: %w", err)
	}

	if len(topicIDs) == 0 {
		return []FocusTopicResult{}, nil
	}

	topicRepo := repo.NewTopicRepository()
	results := make([]FocusTopicResult, 0, len(topicIDs))

	for _, id := range topicIDs {
		topic, err := topicRepo.GetByID(ctx, id)
		if err != nil || topic == nil {
			continue
		}
		results = append(results, FocusTopicResult{
			ID:              topic.ID,
			Name:            topic.Name,
			Category:        topic.Category,
			Source:          topic.Source,
			OccurrenceCount: topic.OccurrenceCount,
			FirstSeenDate:   topic.FirstSeenDate,
			LastSeenDate:    topic.LastSeenDate,
		})
	}

	return results, nil
}

func (s *FocusServiceImpl) Set(ctx context.Context, date string, topicIDs []int64) error {
	focusCache := cache.NewFocusCache()
	if err := focusCache.SetFocusTopics(ctx, date, topicIDs); err != nil {
		return fmt.Errorf("set focus topics: %w", err)
	}
	return nil
}

func (s *FocusServiceImpl) Delete(ctx context.Context, date string, topicID int64) error {
	focusCache := cache.NewFocusCache()
	if err := focusCache.RemoveFocusTopic(ctx, date, topicID); err != nil {
		return fmt.Errorf("remove focus topic: %w", err)
	}
	return nil
}
