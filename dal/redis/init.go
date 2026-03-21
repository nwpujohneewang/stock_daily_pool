package redis

import (
	"context"
	"fmt"
	"stock/config"

	"github.com/redis/go-redis/v9"
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
		fmt.Printf("Warning: failed to connect to redis: %v\n", err)
	} else {
		fmt.Println("Redis connected successfully")
	}
}

func RedisClient(ctx context.Context) *redis.Client {
	return globalClient
}
