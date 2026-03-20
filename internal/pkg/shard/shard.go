package shard

import (
	"context"
	"hash/fnv"

	"golang.org/x/sync/errgroup"
)

func AssignShard(tsCode string, shardCount int) int {
	h := fnv.New32a()
	h.Write([]byte(tsCode))
	return int(h.Sum32()) % shardCount
}

type ShardBatch struct {
	ShardID int
	TsCodes []string
}

func BuildShardBatches(allTsCodes []string, shardCount int) []ShardBatch {
	batches := make([]ShardBatch, shardCount)
	for i := 0; i < shardCount; i++ {
		batches[i] = ShardBatch{ShardID: i}
	}

	for _, code := range allTsCodes {
		shardID := AssignShard(code, shardCount)
		batches[shardID].TsCodes = append(batches[shardID].TsCodes, code)
	}

	return batches
}

func ProcessShards(ctx context.Context, shardCount int, fn func(ctx context.Context, shardID int, tsCodes []string) error) error {
	group, ctx := errgroup.WithContext(ctx)
	for i := 0; i < shardCount; i++ {
		shardID := i
		_ = shardID
		group.Go(func() error {
			return ctx.Err()
		})
	}
	return group.Wait()
}
