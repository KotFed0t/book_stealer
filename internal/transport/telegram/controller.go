package telegram

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"book_stealer_tgbot/config"
	"book_stealer_tgbot/data/session"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/utils"

	tele "gopkg.in/telebot.v4"
)

type BookStealerService interface {
}

type Session interface {
	GetSession(ctx context.Context, key string) (model.Session, error)
	SetSession(ctx context.Context, key string, session model.Session) error
}

type Controller struct {
	cfg                *config.Config
	session            Session
	bookStealerService BookStealerService
}

func NewController(cfg *config.Config, bookStealerService BookStealerService, session Session) *Controller {
	return &Controller{
		cfg:                cfg,
		bookStealerService: bookStealerService,
		session:            session,
	}
}

func (ctrl *Controller) Start(c tele.Context) error {
	ctx := utils.CreateCtxWithRqID(c)
	err := ctrl.bookStealerService.RegUser(context.WithoutCancel(ctx), c.Chat().ID)
	if err != nil {
		return c.Send("Регистрация завершилась с ошибкой. Вызовите команду /start еще раз.")
	}
	return c.Reply("Добро пожаловать! Можешь начать выбрав одну из команд в меню.")
}

func (ctrl *Controller) getSessionFromTeleCtxOrStorage(ctx context.Context, c tele.Context) (model.Session, error) {
	op := "Controller.getSessionFromTeleCtxOrStorage"
	chatSession, ok := c.Get("session").(model.Session)
	if ok {
		return chatSession, nil
	}

	rqID := utils.GetRequestIDFromCtx(ctx)
	chatSession, err := ctrl.session.GetSession(ctx, strconv.FormatInt(c.Chat().ID, 10))
	if err != nil {
		if !errors.Is(err, session.ErrNotFound) {
			slog.Error("got error from session.GetSession", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		}
		return model.Session{}, err
	}
	return chatSession, nil
}

func (ctrl *Controller) sendAutoDeleteMsg(c tele.Context, text string) error {
	msg, err := c.Bot().Send(c.Chat(), text)
	if err != nil {
		return err
	}

	time.AfterFunc(5*time.Second, func() {
		c.Bot().Delete(msg)
	})
	return nil
}
