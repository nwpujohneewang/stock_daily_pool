package scheduler

import (
	"context"

	"stock/dal/repo"
)

func ComputeTopicStockCount(ctx context.Context) error {
	topicStockCountRepo := repo.NewTopicStockCountRepository()
	counts, err := topicStockCountRepo.AggregateFromRelations(ctx)
	if err != nil {
		return err
	}
	return topicStockCountRepo.UpsertBatch(ctx, counts)
}

func WarmupTopicStockCountCache(ctx context.Context) error {
	topicStockCountRepo := repo.NewTopicStockCountRepository()
	return topicStockCountRepo.WarmupCache(ctx)
}
