package repository

import (
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"log/slog"
)

type BotRepo struct {
	db *sqlx.DB
}

func NewBotRepo(db *sqlx.DB) *BotRepo {
	return &BotRepo{db}
}

func (r *BotRepo) GetEmailByChatId(chatId int64) (email string, err error) {
	op := "BotRepo.GetEmailByChatId"

	query := `SELECT email FROM emails WHERE chat_id = $1`
	err = r.db.QueryRowx(query, chatId).Scan(&email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn(
				"No rows in result set for chatId",
				slog.String("op", op),
				slog.String("err", err.Error()),
				slog.Int64("chatId", chatId),
			)
			return "", ErrNoRows
		}
		slog.Error(
			"Failed to get email by chatId",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
		)
		return "", err
	}

	slog.Info(
		"Got email by chatId",
		slog.String("op", op),
		slog.String("email", email),
		slog.Int64("chatId", chatId),
	)
	return email, nil
}

func (r *BotRepo) UpsertEmail(chatId int64, email string) error {
	op := "BotRepo.SetEmail"

	_, err := r.db.Exec(`INSERT INTO emails (chat_id, email) VALUES ($1, $2) ON CONFLICT(chat_id) DO UPDATE SET email = EXCLUDED.email;`, chatId, email)
	if err != nil {
		slog.Error(
			"Failed to upsert email for chatId",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
			slog.String("email", email),
		)
		return err
	}

	slog.Info(
		"Email upserted successfully to DB",
		slog.String("op", op),
		slog.String("email", email),
		slog.Int64("chatId", chatId),
	)
	return nil
}

func (r *BotRepo) DeleteEmailByChatId(chatId int64) error {
	op := "BotRepo.DeleteEmail"
	_, err := r.db.Exec("DELETE FROM emails WHERE chat_id = $1", chatId)
	if err != nil {
		slog.Error(
			"Failed to delete email",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Int64("chatId", chatId),
		)
	}
	return nil
}
