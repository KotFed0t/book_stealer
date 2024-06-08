package controllers

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/lib/files"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/service/botService"
	"book_stealer_tgbot/internal/service/serviceInterface"
	"book_stealer_tgbot/internal/sessions"
	"errors"
	"log/slog"
	"regexp"
)

type BotController struct {
	botService  serviceInterface.IBotService
	scrapper    serviceInterface.IScrapperService
	bookService serviceInterface.IBookService
	session     sessions.ISession
	cfg         *config.Config
}

func NewBotController(
	bs serviceInterface.IBotService,
	scrapper serviceInterface.IScrapperService,
	session sessions.ISession,
	cfg *config.Config,
	bookService serviceInterface.IBookService,
) *BotController {
	return &BotController{botService: bs, scrapper: scrapper, session: session, cfg: cfg, bookService: bookService}
}

func (c *BotController) HandleMessage(chatId int64, msg string) {
	op := "BotController.HandleMessage"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	switch {
	case chatSession.ExpectingAuthor:
		chatSession.Author = msg
		chatSession.ExpectingAuthor = false
		chatSession.CurTgPage = 1
		err = c.session.SetOrUpdateChatSession(chatId, chatSession)
		if err != nil {
			slog.Error(
				"got error while setting chat session",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}

		books, hasNextPage, err := c.bookService.GetBooksForPage(chatId, &chatSession, 1)
		if err != nil {
			slog.Error(
				"got error from bookService.GetBooksForPage",
				slog.String("op", op),
				slog.String("err", err.Error()),
				slog.Any("chatSession", chatSession),
			)
			return
		}

		lastMsgId, err := c.botService.SendBooksForPage(chatId, books, &chatSession, 1, hasNextPage)
		if err != nil {
			slog.Error(
				"got error while send books to telegram",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}

		chatSession.LastMsgId = lastMsgId
		err = c.session.SetOrUpdateChatSession(chatId, chatSession)
		if err != nil {
			slog.Error(
				"got error while setting chat session",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}
	case chatSession.ExpectingEmail:
		err := c.session.DeleteChatSession(chatId)
		if err != nil {
			slog.Error(
				"got error while deleting chat session",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}

		reEmail := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !reEmail.MatchString(msg) {
			_ = c.botService.SendMessage(chatId, "Введен некорректный email. Проверьте правильность и введите команду /email чтобы повторить попытку.")
			return
		}

		err = c.botService.SetEmail(chatId, msg)
		if err != nil {
			slog.Error(
				"got error from botService while setting email",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}

		_ = c.botService.SendMessage(chatId, "Email успешно привязан")
	default:
		//ввели название книги. Все данные из сессии очищаем
		chatSession = model.ChatSession{BookTitle: msg}

		lastMsgId, err := c.botService.SendKeyboardForTitle(chatId, msg)
		if err != nil {
			slog.Error(
				"got error while send keyboard for title to telegram",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}

		chatSession.LastMsgId = lastMsgId
		err = c.session.SetOrUpdateChatSession(chatId, chatSession)
		if err != nil {
			slog.Error(
				"got error while setting chat session",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return
		}
	}
}

func (c *BotController) SetAuthor(chatId int64, author string, msgId int) {
	op := "BotController.SetAuthor"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	chatSession.ExpectingAuthor = true
	err = c.session.SetOrUpdateChatSession(chatId, chatSession)
	if err != nil {
		slog.Error(
			"got error while setting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	err = c.botService.SendKeyboardForAuthor(chatId, author, msgId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
}

func (c *BotController) BackToTitle(chatId int64, msgId int) {
	op := "BotController.BackToTitle"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	chatSession.ExpectingAuthor = false
	err = c.session.SetOrUpdateChatSession(chatId, chatSession)
	if err != nil {
		slog.Error(
			"got error while setting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	_, err = c.botService.SendKeyboardForTitle(chatId, chatSession.BookTitle, msgId)
	if err != nil {
		slog.Error(
			"got error while sending keyboard for title to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
}

func (c *BotController) SearchByBookTitle(chatId int64, msgId int) {
	op := "BotController.SearchByBookTitle"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	chatSession.CurTgPage = 1
	err = c.session.SetOrUpdateChatSession(chatId, chatSession)
	if err != nil {
		slog.Error(
			"got error while setting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	books, hasNextPage, err := c.bookService.GetBooksForPage(chatId, &chatSession, chatSession.CurTgPage)
	if err != nil {
		slog.Error(
			"got error from bookService.GetBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.Any("chatSession", chatSession),
		)
		return
	}

	_, err = c.botService.SendBooksForPage(chatId, books, &chatSession, chatSession.CurTgPage, hasNextPage, msgId)
	if err != nil {
		slog.Error(
			"got error while send books to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
}

func (c *BotController) GetBookData(chatId int64, msgId int, ref string) {
	op := "BotController.GetBookData"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	book, err := c.scrapper.ParseBookPage(ref)
	if err != nil {
		slog.Error(
			"got error while parsing book page",
			slog.String("op", op),
			slog.String("ref", ref),
			slog.String("err", err.Error()),
		)
		return
	}

	slog.Debug("got parsed book page", slog.String("op", op), slog.Any("book", book))

	err = c.botService.SendKeyboardForBook(chatId, book, msgId)
	if err != nil {
		slog.Error(
			"got error while sending message to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
}

func (c *BotController) DownloadBook(chatId int64, msgId int, ref string) {
	op := "BotController.DownloadBook"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...")
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	_ = c.botService.SendMessage(chatId, "Начинаем скачивать книгу...")
	slog.Info("start downloading book", slog.String("op", op), slog.String("ref", ref))
	filePath, err := files.DownloadFile(c.cfg.FilesStorageDir, c.cfg.Flibusta.BaseUrl+ref, c.cfg.ProxyUrl)
	if err != nil {
		slog.Error(
			"got error while downloading file",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так при скачивании книги...")
		return
	}
	slog.Info("book downloaded, start sending to telegram", slog.String("op", op), slog.String("filePath", filePath))
	err = c.botService.SendFile(chatId, filePath)
	if err != nil {
		slog.Error(
			"got error while sending file to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
	slog.Info(
		"book was successfully sent to telegram, start deleting file from local storage",
		slog.String("op", op),
		slog.String("filePath", filePath),
	)

	err = files.DeleteFile(filePath)
	if err != nil {
		slog.Error(
			"got error while deleting file from local storage",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.String("filePath", filePath),
		)
	}
	slog.Info("file deleted from local storage", slog.String("op", op), slog.String("filePath", filePath))
}

func (c *BotController) SendToKindle(chatId int64, msgId int) {
	op := "BotController.SendToKindle"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...")
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	_ = c.botService.SendMessage(chatId, "Запускаем процесс отправки книги (в зависимости от размера файла может занять вплоть до минуты).")
	err = c.botService.SendToKindle(chatId, chatSession.DownloadLinkEpub)
	if err != nil {
		slog.Error(
			"got error while sending book to kindle",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)

		if errors.Is(err, botService.ErrEmailNotFound) {
			_ = c.botService.SendMessage(
				chatId,
				"У вас нет привязанного email, вы можете установить email отправив команду /email боту.",
			)
			return
		}
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...")
		return
	}
	_ = c.botService.SendMessage(
		chatId,
		"Книга успешно отправлена на ваш kindle. Если kindle подключен к wifi - то через несколько минут книга должна отобразиться. "+
			"Возможно придет письмо от Amazon на почту, с подтверждением скачивания книги на kindle. Нужно нажать \"verify request\".",
	)
}

func (c *BotController) HandleCommandEmail(chatId int64) {
	op := "BotController.HandleCommandEmail"
	msgId, err := c.botService.SendKeyboardForEmailCommand(chatId)
	if err != nil {
		slog.Error(
			"got error while sending book to kindle",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	err = c.session.SetOrUpdateChatSession(chatId, model.ChatSession{LastMsgId: msgId})
	if err != nil {
		slog.Error(
			"got error while updating chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
}

func (c *BotController) HandleCommandStart(chatId int64) {
	_ = c.botService.SendMessage(chatId, "Привет! Я могу найти для тебя книгу, просто введи ее название.")
}

func (c *BotController) HandleCommandHelp(chatId int64) {
	_ = c.botService.SendMessage(chatId, "Чтобы найти книгу - просто введи ее название.\n\nЕсли у тебя есть электронная книга от Amazon - то ты можешь привязать свой send-to-kindle email вызвав команду /email и отправлять книги сразу на свою электронную книгу (возможность отправки книги на email появится только если у найденной книги будет предоставлен формат epub).")
}

func (c *BotController) SetOrUpdateEmail(chatId int64, msgId int) {
	op := "BotController.SetOrUpdateEmail"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	chatSession.ExpectingEmail = true
	err = c.session.SetOrUpdateChatSession(chatId, chatSession)
	if err != nil {
		slog.Error(
			"got error while updating chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}
	_ = c.botService.SendMessage(chatId, "Введите ваш send-to-kindle email address. Найти его можно в вашем аккаунте Amazon (content & devices -> preferences -> personal document settings -> Send-to-Kindle E-Mail Settings)", msgId)
}

func (c *BotController) DeleteEmail(chatId int64, msgId int) {
	op := "BotController.DeleteEmail"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	err = c.botService.DeleteEmail(chatId)
	if err != nil {
		slog.Error(
			"got error while deleting email",
			slog.String("op", op),
			slog.Int64("chatId", chatId),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
	}

	_ = c.botService.SendMessage(chatId, "Email успешно удален", msgId)
}

func (c *BotController) NextPage(chatId int64, msgId int) {
	op := "BotController.NextPage"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	chatSession.CurTgPage += 1
	err = c.session.SetOrUpdateChatSession(chatId, chatSession)
	if err != nil {
		slog.Error(
			"got error while setting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	books, hasNextPage, err := c.bookService.GetBooksForPage(chatId, &chatSession, chatSession.CurTgPage)
	if err != nil {
		slog.Error(
			"got error from GetBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	_, err = c.botService.SendBooksForPage(chatId, books, &chatSession, chatSession.CurTgPage, hasNextPage, msgId)
	if err != nil {
		slog.Error(
			"got error from SendBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}
}

func (c *BotController) PrevPage(chatId int64, msgId int) {
	op := "BotController.PrevPage"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	chatSession.CurTgPage -= 1
	err = c.session.SetOrUpdateChatSession(chatId, chatSession)
	if err != nil {
		slog.Error(
			"got error while setting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return
	}

	books, hasNextPage, err := c.bookService.GetBooksForPage(chatId, &chatSession, chatSession.CurTgPage)
	if err != nil {
		slog.Error(
			"got error from GetBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	_, err = c.botService.SendBooksForPage(chatId, books, &chatSession, chatSession.CurTgPage, hasNextPage, msgId)
	if err != nil {
		slog.Error(
			"got error from SendBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}
}

func (c *BotController) BackToBooklist(chatId int64, msgId int) {
	op := "BotController.PrevPage"
	chatSession, err := c.session.GetChatSession(chatId)
	if err != nil {
		slog.Error(
			"got error while getting chat session",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	if chatSession.LastMsgId != msgId {
		_ = c.botService.SendMessage(chatId, "Предыдущие запросы не обрабатываются. Введите новый запрос ;)", msgId)
		return
	}

	books, hasNextPage, err := c.bookService.GetBooksForPage(chatId, &chatSession, chatSession.CurTgPage)
	if err != nil {
		slog.Error(
			"got error from GetBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}

	_, err = c.botService.SendBooksForPage(chatId, books, &chatSession, chatSession.CurTgPage, hasNextPage, msgId)
	if err != nil {
		slog.Error(
			"got error from SendBooksForPage",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		_ = c.botService.SendMessage(chatId, "Что-то пошло не так...", msgId)
		return
	}
}
