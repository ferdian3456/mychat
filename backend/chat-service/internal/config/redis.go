package config

import (
	"context"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strings"
	"time"
)

func NewRedisCluster(config *koanf.Koanf, log *zap.Logger) *redis.ClusterClient {
	redisURLs := config.String("REDIS_URLS")

	addrs := strings.Split(redisURLs, ",")
	
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		DialTimeout:  5 * time.Second, // timeout for establishing new tcp connection to rds
	})

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		log.Fatal("Failed to connect redis", zap.Error(err))
	}

	return rdb
}
