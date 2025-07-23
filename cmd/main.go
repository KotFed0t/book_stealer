package main

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/data/cache"
	"book_stealer_tgbot/data/db/postgres"
	redisClient "book_stealer_tgbot/data/redis"
	"book_stealer_tgbot/data/session"
	"book_stealer_tgbot/internal/downloader"
	"book_stealer_tgbot/internal/externalApi/cloudStorageApi/googleDriveApi"
	"book_stealer_tgbot/internal/mailer"
	"book_stealer_tgbot/internal/parser"
	"book_stealer_tgbot/internal/repository"
	"book_stealer_tgbot/internal/scheduler"
	"book_stealer_tgbot/internal/service/bookStealerService"
	"book_stealer_tgbot/internal/tgbot"
	"book_stealer_tgbot/internal/transport/telegram"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.MustLoad()

	setupLogger(cfg)

	slog.Debug("config", slog.Any("cfg", cfg))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	postgresDb := postgres.NewPostgresClient(cfg)
	defer postgresDb.Close()

	postgresRepo := repository.NewPostgresRepo(postgresDb)

	redisClient := redisClient.MustInitRedis(cfg)
	defer redisClient.Close()

	redisSession := session.NewRedisSession(cfg, redisClient)

	redisCache := cache.NewRedisCache(cfg, redisClient)

	booksParser := parser.NewFlibustaParser(cfg)

	googleCloudStorage := googleDriveApi.New(ctx, cfg)

	fileDownloader := downloader.NewFileDownloader()

	Mailer := mailer.NewMailer(cfg)

	bookStealerService := bookStealerService.New(
		cfg,
		postgresRepo,
		redisCache,
		booksParser,
		googleCloudStorage,
		fileDownloader,
		Mailer,
	)

	sched := scheduler.New()
	sched.NewIntervalJob("delete old files from goolgle drive", googleCloudStorage.DeleteOldFiles, cfg.Jobs.DeleteOldFilesInterval, true)
	sched.Start()
	defer sched.Stop()

	tgController := telegram.NewController(cfg, bookStealerService, redisSession)

	tgBot := tgbot.New(cfg, tgController, redisSession)

	tgBot.Start()
	defer tgBot.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-interrupt
}

func setupLogger(cfg *config.Config) {
	var logLevel slog.Level

	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(log)
}
