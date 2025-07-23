package cache

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewRedisCache(cfg *config.Config, redisClient *redis.Client) *RedisCache {
	return &RedisCache{redis: redisClient, cfg: cfg}
}

func (r *RedisCache) createBooksPageKey(title, author string, page int) string {
	return fmt.Sprintf("title:%s:author:%s:page:%d", title, author, page)
}

func (r *RedisCache) GetBooksForPage(ctx context.Context, title, author string, page int) (booksPage model.BooksPage, err error) {
	op := "RedisCache.GetBooksForPage"
	rqID := utils.GetRequestIDFromCtx(ctx)
	key := r.createBooksPageKey(title, author, page)

	res, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Warn("redis key not found", slog.String("rqID", rqID), slog.String("op", op), slog.String("key", key))
			return model.BooksPage{}, ErrNotFound
		}
		slog.Error("failed on redis.Get", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()), slog.String("key", key))
		return model.BooksPage{}, err
	}

	err = json.Unmarshal([]byte(res), &booksPage)
	if err != nil {
		slog.Error(
			"error while unmarshalling",
			slog.String("rqID", rqID),
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.String("resultFromRedis", res),
		)
		return model.BooksPage{}, errors.New("unmarshalling error")
	}

	return booksPage, nil
}

func (r *RedisCache) SetBooksForPage(ctx context.Context, booksPage model.BooksPage) error {
	op := "RedisCache.SetBooksForPage"
	rqID := utils.GetRequestIDFromCtx(ctx)
	key := r.createBooksPageKey(booksPage.Title, booksPage.Author, booksPage.Page)

	jsonData, err := json.Marshal(booksPage)
	if err != nil {
		slog.Error(
			"error while marshalling",
			slog.String("rqID", rqID),
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Any("booksPage", booksPage),
		)
		return errors.New("marshalling error")
	}

	_, err = r.redis.Set(ctx, key, jsonData, r.cfg.Cache.BooksPageTTL).Result()
	if err != nil {
		slog.Error("failed on redis.Set", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()), slog.Any("booksPage", booksPage))
		return err
	}

	return nil
}
