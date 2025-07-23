package session

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

type RedisSession struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewRedisSession(cfg *config.Config, redisClient *redis.Client) *RedisSession {
	return &RedisSession{redis: redisClient, cfg: cfg}
}

func (r *RedisSession) createSessionKey(chatID int64) string {
	return fmt.Sprintf("chatID:%d:session", chatID)
}

func (r *RedisSession) createBookSearchRequestKey(chatID int64, msgID int) string {
	return fmt.Sprintf("chatID:%d:msgID:%d:request", chatID, msgID)
}

func (r *RedisSession) SetSession(ctx context.Context, chatID int64, session model.Session) error {
	rqID := utils.GetRequestIDFromCtx(ctx)
	slog.Debug("start SetSession", slog.String("rqID", rqID), slog.Any("session", session))

	sessionJson, err := json.Marshal(session)
	if err != nil {
		slog.Error("can't marshall session", slog.String("rqID", rqID), slog.String("err", err.Error()), slog.Any("session", session))
		return errors.New("can't marshall session")
	}

	_, err = r.redis.Set(ctx, r.createSessionKey(chatID), sessionJson, r.cfg.SessionExpiration).Result()
	if err != nil {
		slog.Error("failed on redis.Set", slog.String("rqID", rqID), slog.String("err", err.Error()), slog.Any("session", session))
		return err
	}

	slog.Debug("SetSession completed", slog.String("rqID", rqID))

	return nil
}

func (r *RedisSession) GetSession(ctx context.Context, chatID int64) (model.Session, error) {
	rqID := utils.GetRequestIDFromCtx(ctx)
	slog.Debug("start GetSession", slog.String("rqID", rqID))
	key := r.createSessionKey(chatID)

	res, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Warn("session not found in redis", slog.String("rqID", rqID))
			return model.Session{}, ErrNotFound
		}
		
		slog.Error("failed on redis.Get", slog.String("rqID", rqID), slog.String("err", err.Error()), slog.Any("key", key))
		return model.Session{}, err
	}

	session := model.Session{}

	err = json.Unmarshal([]byte(res), &session)
	if err != nil {
		slog.Error("can't unmarshall session", slog.String("rqID", rqID), slog.String("err", err.Error()), slog.Any("resresultFromRedis", res))
		return model.Session{}, errors.New("can't unmarshall session")
	}

	slog.Debug("GetSession completed", slog.String("rqID", rqID), slog.Any("session", session))

	return session, nil
}

func (r *RedisSession) GetBookSearchRequest(ctx context.Context, chatID int64, msgID int) (request model.BookSearchRequest, err error) {
	op := "RedisSession.GetBookSearchRequest"
	rqID := utils.GetRequestIDFromCtx(ctx)
	key := r.createBookSearchRequestKey(chatID, msgID)

	res, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Warn("redis key not found", slog.String("rqID", rqID), slog.String("op", op), slog.String("key", key))
			return model.BookSearchRequest{}, ErrNotFound
		}
		slog.Error("failed on redis.Get", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()), slog.String("key", key))
		return model.BookSearchRequest{}, err
	}

	err = json.Unmarshal([]byte(res), &request)
	if err != nil {
		slog.Error(
			"error while unmarshalling",
			slog.String("rqID", rqID),
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.String("resultFromRedis", res),
		)
		return model.BookSearchRequest{}, errors.New("unmarshalling error")
	}

	return request, nil
}

func (r *RedisSession) SetBookSearchRequest(ctx context.Context, chatID int64, msgID int, request model.BookSearchRequest) error {
	op := "RedisSession.SetBookSearchRequest"
	rqID := utils.GetRequestIDFromCtx(ctx)
	key := r.createBookSearchRequestKey(chatID, msgID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			"error while marshalling",
			slog.String("rqID", rqID),
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Any("request", request),
		)
		return errors.New("marshalling error")
	}

	_, err = r.redis.Set(ctx, key, jsonData, r.cfg.SessionExpiration).Result()
	if err != nil {
		slog.Error("failed on redis.Set", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()), slog.Any("request", request))
		return err
	}

	return nil
}