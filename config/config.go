package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Env               string        `env:"ENV"`
	LogLevel          string        `env:"LOG_LEVEL"`
	MaxGoroutineCnt   int           `env:"MAX_GOROUTINE_CNT"`
	ProxyUrl          string        `env:"PROXY_URL"`
	BooksPerPage      int           `env:"BOOKS_PER_PAGE"`
	SessionExpiration time.Duration `env:"SESSION_EXPIRATION"`
	Redis             Redis
	Mail              Mail
	Postgres          Postgres
	Telegram          Telegram
	Flibusta          Flibusta
	GoogleDrive       GoogleDrive
	Cache             Cache
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
	MigrationDir    string `env:"PG_MIGRATION_DIR"`
}

type Telegram struct {
	Token      string        `env:"TELEGRAM_TOKEN"`
	UpdTimeout time.Duration `env:"TELEGRAM_UPD_TIMEOUT"`
}

type Flibusta struct {
	BaseUrl      string `env:"FLIBUSTA_BASE_URL"`
	SearchPage   string `env:"FLIBUSTA_SEARCH_PAGE"`
	BooksPerPage int    `env:"FLIBUSTA_BOOKS_PER_PAGE"`
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
	Login    string `env:"MAIL_LOGIN"`
}

type GoogleDrive struct {
	CredentialsFile string        `env:"GOOGLE_DRIVE_CREDENTIALS_FILE"`
	FileTTL         time.Duration `env:"GOOGLE_DRIVE_FILE_TTL"`
}

type Cache struct {
	RequestTTL   time.Duration `env:"CACHE_REQUEST_TTL"`
	BooksPageTTL time.Duration `env:"CACHE_BOOKS_PAGE_TTL"`
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
