package tgbot

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"book_stealer_tgbot/config"
	"book_stealer_tgbot/data/session"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/model/tg/tgCallback.go"
	"book_stealer_tgbot/internal/transport/telegram"
	customMW "book_stealer_tgbot/internal/transport/telegram/middleware"
	"book_stealer_tgbot/utils"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

type Session interface {
	GetSession(ctx context.Context, chatID int64) (model.Session, error)
	SetSession(ctx context.Context, chatID int64, session model.Session) error
}

type TGBot struct {
	bot     *tele.Bot
	ctrl    *telegram.Controller
	session Session
}

func New(cfg *config.Config, ctrl *telegram.Controller, session Session) *TGBot {
	settings := tele.Settings{
		Token:  cfg.Telegram.Token,
		Poller: &tele.LongPoller{Timeout: cfg.Telegram.UpdTimeout},
	}

	b, err := tele.NewBot(settings)
	if err != nil {
		slog.Error("error while tele.NewBot", slog.String("err", err.Error()))
		panic(err)
	}

	return &TGBot{bot: b, ctrl: ctrl, session: session}
}

func (b *TGBot) Start() {
	b.bot.Use(middleware.Recover(), customMW.Logger())

	b.setupRoutes()

	go b.bot.Start()
	slog.Info("tgbot started!")
}

func (b *TGBot) Stop() {
	slog.Info("start stopping tgbot")
	b.bot.Stop()
	slog.Info("tgbot stopped")
}

func (b *TGBot) setupRoutes() {
	// commands
	b.bot.Handle("/start", b.ctrl.Start)
	b.bot.Handle("/help", b.ctrl.Help)
	b.bot.Handle("/email", b.ctrl.Email)

	// text
	b.bot.Handle(tele.OnText, func(c tele.Context) error {
		// получение сесии и выбор метода контроллера на основе шага пользователя
		ctx := utils.CreateCtxWithRqID(c)
		rqID := utils.GetRequestIDFromCtx(ctx)
		chatSession, err := b.session.GetSession(ctx, c.Chat().ID)
		if err != nil && !errors.Is(err, session.ErrNotFound) {
			slog.Error("got error from session.GetSession", slog.String("rqID", rqID), slog.String("err", err.Error()))
			return c.Send("что-то пошло не так...")
		}

		c.Set("session", chatSession)

		switch chatSession.Action {
		case model.ExpectingAuthor:
			return b.ctrl.ProcessEnterAuthorSurname(c)
		case model.ExpectingEmail:
			return b.ctrl.ProcessLinkEmail(c)
		default:
			return b.ctrl.ProcessEnteredTitle(c)
		}
	})

	// callbacks
	b.bot.Handle(tele.OnCallback, func(c tele.Context) error {
		callbackBtnText := strings.TrimPrefix(c.Callback().Data, "\f")

		switch {
		case callbackBtnText == tgCallback.SearchByBookTitle:
			return b.ctrl.SearchByBookTitle(c)
		case callbackBtnText == tgCallback.EnterAuthorSurname:
			return b.ctrl.InitEnterAuthorSurname(c)
		case callbackBtnText == tgCallback.BackToBooksPage:
			return b.ctrl.BackToBooksPage(c)
		case callbackBtnText == tgCallback.LinkEmail:
			return b.ctrl.InitLinkEmail(c)
		case callbackBtnText == tgCallback.DeleteEmail:
			return b.ctrl.DeleteEmail(c)
		case strings.HasPrefix(callbackBtnText, tgCallback.ToBooksPage):
			return b.ctrl.ProcessToBooksPage(c)
		case strings.HasPrefix(callbackBtnText, tgCallback.DownloadBook):
			return b.ctrl.DownloadBook(c)
		case strings.HasPrefix(callbackBtnText, tgCallback.ToBookDetails):
			return b.ctrl.ProcessToBookDetails(c)
		case strings.HasPrefix(callbackBtnText, tgCallback.SendToKindle):
			return b.ctrl.SendBookToKindle(c)
		case callbackBtnText == tgCallback.PageNumber:
			return nil
		default:
			return c.Send("callback не опознан")
		}
	})

}
