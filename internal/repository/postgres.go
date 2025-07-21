package repository

import (
	"book_stealer_tgbot/utils"
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	db *sqlx.DB
}

func NewPostgresRepo(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (r *Postgres) GetEmailByChatId(ctx context.Context, chatId int64) (email string, err error) {
	op := "Postgres.GetEmailByChatId"
	rqID := utils.GetRequestIDFromCtx(ctx)
	query := `SELECT email FROM emails WHERE chat_id = $1`

	err = r.db.QueryRowxContext(ctx, query, chatId).Scan(&email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn(
				"No rows in result set for chatId",
				slog.String("op", op),
				slog.String("rqID", rqID),
				slog.String("err", err.Error()),
				slog.Int64("chatId", chatId),
			)
			return "", ErrNoRows
		}
		slog.Error(
			"Failed to get email by chatId",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
		)
		return "", err
	}

	slog.Info(
		"Got email by chatId",
		slog.String("op", op),
		slog.String("rqID", rqID),
		slog.String("email", email),
		slog.Int64("chatId", chatId),
	)
	return email, nil
}

func (r *Postgres) UpsertEmail(ctx context.Context, chatId int64, email string) error {
	op := "Postgres.UpsertEmail"
	rqID := utils.GetRequestIDFromCtx(ctx)
	query := `INSERT INTO emails (chat_id, email) VALUES ($1, $2) ON CONFLICT(chat_id) DO UPDATE SET email = EXCLUDED.email;`

	_, err := r.db.ExecContext(ctx, query, chatId, email)
	if err != nil {
		slog.Error(
			"Failed to upsert email for chatId",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
			slog.String("email", email),
		)
		return err
	}

	slog.Info(
		"Email upserted successfully to DB",
		slog.String("op", op),
		slog.String("rqID", rqID),
		slog.String("email", email),
		slog.Int64("chatId", chatId),
	)
	return nil
}

func (r *Postgres) DeleteEmailByChatId(ctx context.Context, chatId int64) error {
	op := "Postgres.DeleteEmail"
	rqID := utils.GetRequestIDFromCtx(ctx)
	query := `DELETE FROM emails WHERE chat_id = $1`

	_, err := r.db.ExecContext(ctx, query, chatId)
	if err != nil {
		slog.Error(
			"Failed to delete email",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
		)
	}
	return nil
}
