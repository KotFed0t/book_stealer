package cache

import (
	"book_stealer_tgbot/config"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewRedisCache(cfg *config.Config, redisClient *redis.Client) *RedisCache {
	return &RedisCache{redis: redisClient, cfg: cfg}
}
