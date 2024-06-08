package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"log"
)

type Config struct {
	Env             string `env:"ENV"`
	LogLevel        string `env:"LOG_LEVEL"`
	MaxGoroutineCnt int    `env:"MAX_GOROUTINE_CNT"`
	FilesStorageDir string `env:"FILES_STORAGE_DIR"`
	Postgres        Postgres
	Telegram        Telegram
	Flibusta        Flibusta
	ProxyUrl        string `env:"PROXY_URL"`
	Redis           Redis
	Mail            Mail
	BooksPerPage    int `env:"BOOKS_PER_PAGE"`
}

type Postgres struct {
	Host            string `env:"PG_HOST"`
	Port            int    `env:"PG_PORT"`
	DbName          string `env:"PG_DB_NAME"`
	Password        string `env:"PG_PASSWORD"`
	User            string `env:"PG_USER"`
	PoolMax         int    `env:"PG_POOL_MAX"`
	MaxOpenConns    int    `env:"PG_MAX_OPEN_CONNS"`
	ConnMaxLifetime int    `env:"PG_CONN_MAX_LIFETIME"`
	MaxIdleConns    int    `env:"PG_MAX_IDLE_CONNS"`
	ConnMaxIdleTime int    `env:"PG_CONN_MAX_IDLE_TIME"`
}

type Telegram struct {
	Token      string `env:"TELEGRAM_TOKEN"`
	UpdTimeout int    `env:"TELEGRAM_UPD_TIMEOUT"`
}

type Flibusta struct {
	BaseUrl    string `env:"FLIBUSTA_BASE_URL"`
	SearchPage string `env:"FLIBUSTA_SEARCH_PAGE"`
}

type Redis struct {
	Host     string `env:"REDIS_HOST"`
	Port     int    `env:"REDIS_PORT"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB"`
}

type Mail struct {
	Host     string `env:"MAIL_HOST"`
	Port     int    `env:"MAIL_PORT"`
	Address  string `env:"MAIL_ADDRESS"`
	Password string `env:"MAIL_PASSWORD"`
}

func MustLoad() *Config {
	_ = godotenv.Load(".env")

	cfg := &Config{}

	opts := env.Options{RequiredIfNoDef: true}

	if err := env.ParseWithOptions(cfg, opts); err != nil {
		log.Fatalf("parse config error: %s", err)
	}

	return cfg
}
