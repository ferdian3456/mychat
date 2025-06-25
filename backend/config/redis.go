package config

import (
	"context"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRedis(config *koanf.Koanf, log *zap.Logger) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: config.String("REDIS_URL"),
	})

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		log.Fatal("Failed to connect to redis", zap.Error(err))
	}

	return rdb
}
