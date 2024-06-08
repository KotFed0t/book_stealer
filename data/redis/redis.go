package redis

import (
	"book_stealer_tgbot/config"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

func MustInitRedis(cfg *config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		slog.Error("Error while connecting Redis", slog.String("error", err.Error()))
		panic(err)
	}
	slog.Info("Redis connected", slog.String("pong", pong))

	return rdb
}
