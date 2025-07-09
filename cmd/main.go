package main

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/data/cache"
	"book_stealer_tgbot/data/db/postgres"
	redisClient "book_stealer_tgbot/data/redis"
	"book_stealer_tgbot/data/session"
	"book_stealer_tgbot/internal/controllers"
	"book_stealer_tgbot/internal/externalApi/cloudStorageApi/googleDriveApi"
	"book_stealer_tgbot/internal/parser"
	"book_stealer_tgbot/internal/repository"
	"book_stealer_tgbot/internal/service/bookStealerService"
	"book_stealer_tgbot/internal/tgbot"
	"book_stealer_tgbot/internal/transport/telegram"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	bookStealerService := bookStealerService.New(cfg, postgresRepo, redisCache, booksParser, googleCloudStorage)

	tgController := telegram.NewController(cfg, bookStealerService, redisSession)

	tgBot := tgbot.New(cfg, tgController, redisSession)

	tgBot.Start()
	defer tgBot.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-interrupt
}

func handleUpdates(updates <-chan tgbotapi.Update, botController *controllers.BotController, wg *sync.WaitGroup) {
	slog.Info("goroutine started, waiting for updates")
	for update := range updates {
		slog.Info("goroutine received update, start handling", slog.Any("update", update))
		if update.Message != nil {
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					botController.HandleCommandStart(update.Message.Chat.ID)
				case "help":
					botController.HandleCommandHelp(update.Message.Chat.ID)
				case "email":
					botController.HandleCommandEmail(update.Message.Chat.ID)
				}
			} else {
				botController.HandleMessage(update.Message.Chat.ID, update.Message.Text)
			}
		} else if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "set_author":
				botController.SetAuthor(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.Text,
					update.CallbackQuery.Message.MessageID,
				)

			case "back_to_title":
				botController.BackToTitle(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)

			case "search_by_book_title":
				botController.SearchByBookTitle(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)

			case "send_to_kindle":
				botController.SendToKindle(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)

			case "set_or_update_email":
				botController.SetOrUpdateEmail(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)

			case "delete_email":
				botController.DeleteEmail(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)

			case "next_page":
				botController.NextPage(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)

			case "prev_page":
				botController.PrevPage(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)

			case "back_to_booklist":
				botController.BackToBooklist(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
				)
			default:
				reBook := regexp.MustCompile(`^/b/\d+$`)
				reBookDownload := regexp.MustCompile(`^/b/\d+/\w+$`)

				if reBook.MatchString(update.CallbackQuery.Data) {
					botController.GetBookData(
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID,
						update.CallbackQuery.Data,
					)
				} else if reBookDownload.MatchString(update.CallbackQuery.Data) {
					botController.DownloadBook(
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID,
						update.CallbackQuery.Data,
					)
				}
			}
		}
	}
	slog.Info("Channel updates was closed, exit from goroutine")
	wg.Done()
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
