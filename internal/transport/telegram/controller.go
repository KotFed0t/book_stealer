package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"book_stealer_tgbot/config"
	"book_stealer_tgbot/data/session"
	"book_stealer_tgbot/internal/converter/telebotConverter"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/model/tg/tgCallback.go"
	"book_stealer_tgbot/internal/service"
	"book_stealer_tgbot/utils"

	tele "gopkg.in/telebot.v4"
)

type BookStealerService interface {
	GetBooksForPage(ctx context.Context, request model.BookSearchRequest) (booksPage model.BooksPage, err error)
	GetBookDetails(ctx context.Context, bookLink string) (book model.Book, err error)
}

type Session interface {
	GetSession(ctx context.Context, chatID int64) (model.Session, error)
	SetSession(ctx context.Context, chatID int64, session model.Session) error
	GetBookSearchRequest(ctx context.Context, chatID int64, msgID int) (request model.BookSearchRequest, err error)
	SetBookSearchRequest(ctx context.Context, chatID int64, msgID int, request model.BookSearchRequest) error
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

func (ctrl *Controller) getSessionFromTeleCtxOrStorage(ctx context.Context, c tele.Context) (model.Session, error) {
	op := "Controller.getSessionFromTeleCtxOrStorage"
	chatSession, ok := c.Get("session").(model.Session)
	if ok {
		return chatSession, nil
	}

	rqID := utils.GetRequestIDFromCtx(ctx)
	chatSession, err := ctrl.session.GetSession(ctx, c.Chat().ID)
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

func (ctrl *Controller) Start(c tele.Context) error {
	return c.Reply("Добро пожаловать! Я могу найти для тебя книгу, просто введи ее название (фамилию автора можно будет указать позже).")
}

func (ctrl *Controller) Help(c tele.Context) error {
	return c.Reply("Чтобы найти книгу - просто введи ее название.\n\nЕсли у тебя есть электронная книга от Amazon - то ты можешь привязать свой send-to-kindle email вызвав команду /email и отправлять книги сразу на свою электронную книгу (возможность отправки книги на email появится только если у найденной книги будет предоставлен формат epub).\n\nВажно! Чтобы книги успешно приходили на kindle добавь email booksender@kotfedot-projects.ru в Approved Personal Document E-mail List. Для этого зайди в аккаунт Amazon (content & devices -> preferences -> personal document settings -> Approved Personal Document E-mail List)")
}

func (ctrl *Controller) ProcessEnteredTitle(c tele.Context) error {
	op := "Controller.ProcessEnteredTitle"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)
	title := c.Message().Text

	chatSession := model.Session{BookTitle: title}
	err := ctrl.session.SetSession(ctx, c.Chat().ID, chatSession)
	if err != nil {
		slog.Error("got error from session.SetSession", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	return c.Send(telebotConverter.EnteredTitleMenuResponse(title))
}

func (ctrl *Controller) SearchByBookTitle(c tele.Context) error {
	op := "Controller.SearchByBookTitle"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)

	chatSession, err := ctrl.getSessionFromTeleCtxOrStorage(ctx, c)
	if err != nil {
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	request := model.BookSearchRequest{
		Title:  chatSession.BookTitle,
		Author: "",
		Page:   0,
	}

	booksPage, err := ctrl.bookStealerService.GetBooksForPage(ctx, request)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.Warn(
				"books not found",
				slog.String("rqID", rqID),
				slog.String("op", op),
				slog.String("err", err.Error()),
				slog.String("title", chatSession.BookTitle),
				slog.String("author", chatSession.Author),
			)

			return c.Edit(telebotConverter.BooksNotFound(chatSession.BookTitle, chatSession.Author))
		}
		slog.Error("got error from bookStealerService.GetBooksForPage", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	go ctrl.session.SetBookSearchRequest(context.WithoutCancel(ctx), c.Chat().ID, c.Message().ID, request)

	return c.Edit(telebotConverter.BooksPage(booksPage, ctrl.cfg.BooksPerPage))
}

func (ctrl *Controller) InitEnterAuthorSurname(c tele.Context) error {
	op := "Controller.InitEnterAuthorSurname"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)

	chatSession, err := ctrl.getSessionFromTeleCtxOrStorage(ctx, c)
	if err != nil {
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	chatSession.Action = model.ExpectingAuthor
	err = ctrl.session.SetSession(ctx, c.Chat().ID, chatSession)
	if err != nil {
		slog.Error("got error from session.SetSession", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	return c.Edit(telebotConverter.EnterAuthorResponse())
}

func (ctrl *Controller) ProcessEnterAuthorSurname(c tele.Context) error {
	op := "Controller.ProcessEnterAuthorSurname"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)

	chatSession, err := ctrl.getSessionFromTeleCtxOrStorage(ctx, c)
	if err != nil {
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	chatSession.Author = c.Message().Text
	chatSession.Action = model.DefaultAction
	err = ctrl.session.SetSession(ctx, c.Chat().ID, chatSession)
	if err != nil {
		slog.Error("got error from session.SetSession", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	request := model.BookSearchRequest{
		Title:  chatSession.BookTitle,
		Author: chatSession.Author,
		Page:   0,
	}

	booksPage, err := ctrl.bookStealerService.GetBooksForPage(ctx, request)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.Warn(
				"books not found",
				slog.String("rqID", rqID),
				slog.String("op", op),
				slog.String("err", err.Error()),
				slog.String("title", chatSession.BookTitle),
				slog.String("author", chatSession.Author),
			)

			return c.Send(telebotConverter.BooksNotFound(chatSession.BookTitle, chatSession.Author))
		}
		slog.Error("got error from bookStealerService.GetBooksForPage", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return c.Send(internalErrMsg)
	}

	text, markup := telebotConverter.BooksPage(booksPage, ctrl.cfg.BooksPerPage)

	msg, err := c.Bot().Send(c.Recipient(), text, markup)
	if err == nil {
		go ctrl.session.SetBookSearchRequest(context.WithoutCancel(ctx), c.Chat().ID, msg.ID, request)
	}

	return err
}

func (ctrl *Controller) ProcessToBooksPage(c tele.Context) error {
	op := "Controller.ProcessToBooksPage"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)

	pageStr := strings.TrimPrefix(c.Callback().Data, fmt.Sprintf("\f%s", tgCallback.ToBooksPage))
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		slog.Error(
			"error while converting page from callback",
			slog.String("rqID", rqID),
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.String("pageStr", pageStr),
		)
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	request, err := ctrl.session.GetBookSearchRequest(ctx, c.Chat().ID, c.Message().ID)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return ctrl.sendAutoDeleteMsg(c, requestTooOld)
		}
		slog.Error("got error from session.GetBookSearchRequest", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	request.Page = page

	booksPage, err := ctrl.bookStealerService.GetBooksForPage(ctx, request)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.Warn(
				"books not found",
				slog.String("rqID", rqID),
				slog.String("op", op),
				slog.String("err", err.Error()),
				slog.String("title", request.Title),
				slog.String("author", request.Author),
			)
			return ctrl.sendAutoDeleteMsg(c, booksNotFound)
		}
		slog.Error("got error from bookStealerService.GetBooksForPage", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	go ctrl.session.SetBookSearchRequest(context.WithoutCancel(ctx), c.Chat().ID, c.Message().ID, request)

	return c.Edit(telebotConverter.BooksPage(booksPage, ctrl.cfg.BooksPerPage))
}

func (ctrl *Controller) ProcessToBookDetails(c tele.Context) error {
	op := "Controller.ProcessToBookDetails"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)

	bookLink := strings.TrimPrefix(c.Callback().Data, fmt.Sprintf("\f%s", tgCallback.ToBookDetails))

	book, err := ctrl.bookStealerService.GetBookDetails(ctx, bookLink)
	if err != nil {
		slog.Error("got error from bookStealerService.GetBookDetails", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	return c.Edit(telebotConverter.BookDetails(book))
}
func (ctrl *Controller) BackToBooksPage(c tele.Context) error {
	op := "Controller.BackToBooksPage"
	ctx := utils.CreateCtxWithRqID(c)
	rqID := utils.GetRequestIDFromCtx(ctx)

	request, err := ctrl.session.GetBookSearchRequest(ctx, c.Chat().ID, c.Message().ID)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return ctrl.sendAutoDeleteMsg(c, requestTooOld)
		}
		slog.Error("got error from session.GetBookSearchRequest", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	booksPage, err := ctrl.bookStealerService.GetBooksForPage(ctx, request)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.Warn(
				"books not found",
				slog.String("rqID", rqID),
				slog.String("op", op),
				slog.String("err", err.Error()),
				slog.String("title", request.Title),
				slog.String("author", request.Author),
			)
			return ctrl.sendAutoDeleteMsg(c, booksNotFound)
		}
		slog.Error("got error from bookStealerService.GetBooksForPage", slog.String("rqID", rqID), slog.String("op", op), slog.String("err", err.Error()))
		return ctrl.sendAutoDeleteMsg(c, internalErrMsg)
	}

	return c.Edit(telebotConverter.BooksPage(booksPage, ctrl.cfg.BooksPerPage))
}
