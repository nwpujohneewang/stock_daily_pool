package redis

import (
	"context"
	"fmt"
	"stock/config"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func Init() {
	c := config.GlobalConfig.Redis
	Client = redis.NewClient(&redis.Options{
		Addr:     c.Address,
		Password: c.Password,
		DB:       c.DB,
	})

	ctx := context.Background()
	if err := Client.Ping(ctx).Err(); err != nil {
		fmt.Printf("Warning: failed to connect to redis: %v\n", err)
	} else {
		fmt.Println("Redis connected successfully")
	}
}
