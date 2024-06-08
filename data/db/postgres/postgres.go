package postgres

import (
	"book_stealer_tgbot/config"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/stdlib" // pgx driver
	"github.com/jmoiron/sqlx"
)

const (
	defaultConnAttemts = 10
	connTimeout        = time.Second
)

func MustInitPostgres(c *config.Config) *sqlx.DB {
	dataSourceName := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s",
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.User,
		c.Postgres.DbName,
		c.Postgres.Password,
	)

	connAttempts := defaultConnAttemts
	var db *sqlx.DB
	var err error

	for connAttempts > 0 {
		db, err = sqlx.Connect("pgx", dataSourceName)
		if err == nil {
			break
		}

		slog.Info("Postgres is trying to connect, attempts left: %d", connAttempts)

		time.Sleep(connTimeout)

		connAttempts--
	}

	if err != nil {
		slog.Error(fmt.Sprintf("Postgres connAttempts = 0"))
		panic(err)
	}

	db.SetMaxOpenConns(c.Postgres.MaxOpenConns)
	db.SetConnMaxLifetime(time.Duration(c.Postgres.ConnMaxLifetime) * time.Second)
	db.SetMaxIdleConns(c.Postgres.MaxIdleConns)
	db.SetConnMaxIdleTime(time.Duration(c.Postgres.ConnMaxIdleTime) * time.Second)
	if err = db.Ping(); err != nil {
		slog.Error(fmt.Sprintf("Postgres dbPing error"))
		panic(err)
	}

	return db
}
