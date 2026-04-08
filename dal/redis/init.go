package redis

import (
	"context"
	"stock/config"
	"stock/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var globalClient *redis.Client

func Init() {
	c := config.GlobalConfig.Redis
	globalClient = redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	})

	ctx := context.Background()
	if err := globalClient.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to connect to redis", zap.Error(err))
	} else {
		logger.Info("redis connected successfully")
	}
}

func RedisClient(ctx context.Context) *redis.Client {
	return globalClient
}
