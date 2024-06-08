package sessions

import (
	"book_stealer_tgbot/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"strconv"
	"time"
)

type RedisSession struct {
	redis *redis.Client
}

func NewRedisSession(redisClient *redis.Client) *RedisSession {
	return &RedisSession{redis: redisClient}
}

func (r *RedisSession) GetChatSession(chatId int64) (model.ChatSession, error) {
	op := "RedisSession.GetChatSession"
	ctx := context.Background()
	result, err := r.redis.Get(ctx, strconv.FormatInt(chatId, 10)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Info(
				"There is no session in Redis for chatId",
				slog.String("chatId", strconv.FormatInt(chatId, 10)),
				slog.String("op", op),
			)
			return model.ChatSession{}, nil
		}
		slog.Error("error getting session info from Redis", slog.String("op", op), slog.String("err", err.Error()))
		return model.ChatSession{}, fmt.Errorf("%s: error getting session info from Redis - %w", op, err)
	}

	slog.Info(
		"Got ChatSession from Redis for chatId",
		slog.String("chatId", strconv.FormatInt(chatId, 10)),
	)

	var chatSession model.ChatSession
	err = json.Unmarshal([]byte(result), &chatSession)
	if err != nil {
		slog.Error("error while unmarshall session info", slog.String("op", op), slog.String("err", err.Error()))
		return model.ChatSession{}, fmt.Errorf("%s: error while unmarshall session info - %w", op, err)
	}
	return chatSession, nil
}

func (r *RedisSession) SetOrUpdateChatSession(chatId int64, chatSession model.ChatSession) error {
	op := "RedisSession.SetOrUpdateChatSession"
	ctx := context.Background()
	jsonChatSession, err := json.Marshal(chatSession)
	if err != nil {
		slog.Error(
			"error while marshall chatSession info",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("%s: error while marshall chatSession info - %w", op, err)
	}
	err = r.redis.Set(ctx, strconv.FormatInt(chatId, 10), string(jsonChatSession), 2*time.Hour).Err()
	if err != nil {
		slog.Error(
			"Error while setting chatSession info in Redis",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return fmt.Errorf("%s: Error while seting chatSession info in Redis - %w", op, err)
	}

	slog.Info(
		"set or updated ChatSession in Redis for chatId",
		slog.String("chatId", strconv.FormatInt(chatId, 10)),
	)
	return nil
}

func (r *RedisSession) DeleteChatSession(chatId int64) error {
	op := "RedisSession.DeleteChatSession"
	ctx := context.Background()
	err := r.redis.Del(ctx, strconv.FormatInt(chatId, 10)).Err()
	if err != nil {
		slog.Error(
			"Error while deleting chatSession info in Redis",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
		)
		return err
	}
	return nil
}
