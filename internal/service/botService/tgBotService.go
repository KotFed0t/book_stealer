package botService

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/lib/files"
	"book_stealer_tgbot/internal/lib/mail"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/repository"
	"book_stealer_tgbot/internal/sessions"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"strconv"
	"strings"
)

type TgBotService struct {
	repo    repository.IRepository
	session sessions.ISession
	bot     *tgbotapi.BotAPI
	cfg     *config.Config
}

func NewTgBotService(repo repository.IRepository, session sessions.ISession, bot *tgbotapi.BotAPI, cfg *config.Config) *TgBotService {
	return &TgBotService{repo: repo, session: session, bot: bot, cfg: cfg}
}

func (s *TgBotService) SendMessage(chatId int64, msg string, msgId ...int) error {
	op := "TgBotService.SendMessage"

	if len(msgId) != 0 {
		_, err := s.bot.Send(tgbotapi.NewEditMessageText(chatId, msgId[0], msg))
		if err != nil {
			slog.Error(
				"error while send message to telegram",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return err
		}
		return nil
	}

	_, err := s.bot.Send(tgbotapi.NewMessage(chatId, msg))
	if err != nil {
		slog.Error(
			"error while send message to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}

	return nil
}

func (s *TgBotService) SendBooksForPage(
	chatId int64,
	books []model.BookPreview,
	chatSession *model.ChatSession,
	page int,
	hasNextPage bool,
	msgId ...int,
) (int, error) {
	op := "TgBotService.SendBooks"
	if len(books) == 0 {
		return 0, ErrBooksAreEmpty
	}

	booksMessage := fmt.Sprintf("Результаты поиска: %s %s\n\n", chatSession.BookTitle, chatSession.Author)
	cnt := (page - 1) * s.cfg.BooksPerPage
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, book := range books {
		cnt++
		booksMessage += fmt.Sprintf("%d) %s \n\n", cnt, book.Title)
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(cnt), book.Link))
		if cnt%5 == 0 || i == len(books)-1 {
			rows = append(rows, row)
			row = nil
		}
	}

	var pagesRow []tgbotapi.InlineKeyboardButton

	if page > 1 {
		pagesRow = append(pagesRow, tgbotapi.NewInlineKeyboardButtonData("назад", "prev_page"))
	}
	pagesRow = append(pagesRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("страница %d", page), "cur_page"))
	if hasNextPage {
		pagesRow = append(pagesRow, tgbotapi.NewInlineKeyboardButtonData("вперед", "next_page"))
	}

	rows = append(rows, pagesRow)

	// если передан msgId для апдейта
	if len(msgId) != 0 {
		msg := tgbotapi.NewEditMessageTextAndMarkup(chatId, msgId[0], booksMessage, tgbotapi.NewInlineKeyboardMarkup(rows...))
		res, err := s.bot.Send(msg)
		if err != nil {
			slog.Error(
				"error while send message with inline keyboard to telegram",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return 0, err
		}
		return res.MessageID, nil
	}

	msg := tgbotapi.NewMessage(chatId, booksMessage)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	res, err := s.bot.Send(msg)
	if err != nil {
		slog.Error(
			"error while send message with inline keyboard to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return 0, err
	}

	return res.MessageID, nil
}

func (s *TgBotService) SendKeyboardForTitle(chatId int64, bookTitle string, msgId ...int) (int, error) {
	op := "TgBotService.SendKeyboardForTitle"

	titleKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("указать фамилию автора", "set_author"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("искать по названию книги", "search_by_book_title"),
		),
	)

	//если передан msgId для апдейта сообщения
	if len(msgId) != 0 {
		msg := tgbotapi.NewEditMessageTextAndMarkup(
			chatId,
			msgId[0],
			fmt.Sprintf("Вы ввели название книги: %s", bookTitle),
			titleKeyboard,
		)

		res, err := s.bot.Send(msg)
		if err != nil {
			slog.Error(
				"error while send update message with inline keyboard to telegram",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return 0, err
		}

		return res.MessageID, nil
	}

	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Вы ввели название книги: %s", bookTitle))
	msg.ReplyMarkup = titleKeyboard
	res, err := s.bot.Send(msg)
	if err != nil {
		slog.Error(
			"error while send message with inline keyboard to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return 0, err
	}
	return res.MessageID, nil
}

func (s *TgBotService) SendKeyboardForAuthor(chatId int64, author string, msgId int) error {
	op := "TgBotService.SendKeyboardForAuthor"
	authorKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("назад", "back_to_title"),
		),
	)
	msg := tgbotapi.NewEditMessageTextAndMarkup(chatId, msgId, "Введите фамилию автора", authorKeyboard)
	if _, err := s.bot.Send(msg); err != nil {
		slog.Error(
			"error while send message with inline keyboard to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}
	return nil
}

func (s *TgBotService) SendKeyboardForBook(chatId int64, book model.Book, msgId int) error {
	op := "TgBotService.SendKeyboardForBook"
	downloadButtons := []tgbotapi.InlineKeyboardButton{}
	sendToKindleButton := []tgbotapi.InlineKeyboardButton{}
	if len(book.DownloadRefs) > 0 {
		for ref, text := range book.DownloadRefs {
			downloadButtons = append(downloadButtons, tgbotapi.NewInlineKeyboardButtonData(text, ref))
			if strings.Contains(text, "(epub)") || strings.Contains(text, "(скачать epub)") {
				chatSession, err := s.session.GetChatSession(chatId)
				if err != nil {
					slog.Error(
						"error while getting session",
						slog.String("op", op),
						slog.String("err", err.Error()),
					)
					return err
				}
				chatSession.DownloadLinkEpub = ref
				err = s.session.SetOrUpdateChatSession(chatId, chatSession)
				if err != nil {
					slog.Error(
						"error while getting session",
						slog.String("op", op),
						slog.String("err", err.Error()),
					)
					return err
				}

				sendToKindleButton = append(sendToKindleButton, tgbotapi.NewInlineKeyboardButtonData("отправить на kindle", "send_to_kindle"))
			}
		}
	}
	bookKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("назад", "back_to_booklist")),
		downloadButtons,
		sendToKindleButton,
	)

	var authors string
	for id, autor := range book.Authors {
		if id > 0 {
			authors += ", " + autor
		} else {
			authors += autor
		}
	}

	text := fmt.Sprintf("%s \n\nАвторство: %s\n\nО книге:\n%s", book.Title, authors, book.Annotation)

	msg := tgbotapi.NewEditMessageTextAndMarkup(chatId, msgId, text, bookKeyboard)
	if _, err := s.bot.Send(msg); err != nil {
		slog.Error(
			"error while send message with inline keyboard to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}
	return nil
}

func (s *TgBotService) SendFile(chatId int64, filePath string) error {
	op := "TgBotService.SendFile"
	file := tgbotapi.FilePath(filePath)

	msg := tgbotapi.DocumentConfig{
		BaseFile: tgbotapi.BaseFile{File: file},
	}

	msg.ChatID = chatId

	_, err := s.bot.Send(msg)
	if err != nil {
		slog.Error(
			"got error while sending file to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}
	return nil
}

func (s *TgBotService) SendToKindle(chatId int64, downloadLink string) error {
	op := "TgBotService.SendToKindle"
	email, err := s.repo.GetEmailByChatId(chatId)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			slog.Warn(
				"Not found email by chatId in DB",
				slog.String("op", op),
				slog.Int64("chatId", chatId),
			)
			return ErrEmailNotFound
		}

		slog.Error(
			"got error while getting email by chatId from Repo",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}

	slog.Info(
		"start downloading file",
		slog.String("op", op),
		slog.Int64("chatId", chatId),
		slog.String("downloadLink", downloadLink),
	)
	filePath, err := files.DownloadFile(s.cfg.FilesStorageDir, s.cfg.Flibusta.BaseUrl+downloadLink, s.cfg.ProxyUrl)
	if err != nil {
		slog.Error(
			"got error while downloading file",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}

	slog.Info(
		"file downloaded, start sending file to email",
		slog.String("op", op),
		slog.Int64("chatId", chatId),
		slog.String("filePath", filePath),
	)

	err = mail.SendFile(s.cfg, filePath, email)
	if err != nil {
		slog.Error(
			"got error while sending file to email",
			slog.String("op", op),
			slog.String("err", err.Error()),
			slog.String("email", email),
		)
		return err
	}

	slog.Info(
		"file sent to email successfully",
		slog.String("op", op),
		slog.Int64("chatId", chatId),
		slog.String("filePath", filePath),
		slog.String("email", email),
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
	return nil
}

func (s *TgBotService) SendKeyboardForEmailCommand(chatId int64) (int, error) {
	op := "TgBotService.SendKeyboardForEmailCommand"
	var keyboard tgbotapi.InlineKeyboardMarkup
	email, err := s.repo.GetEmailByChatId(chatId)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			slog.Warn(
				"Not found email by chatId in DB",
				slog.String("op", op),
				slog.Int64("chatId", chatId),
			)
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("привязать", "set_or_update_email"),
				),
			)
			msg := tgbotapi.NewMessage(chatId, "У вас нет привязанного email")
			msg.ReplyMarkup = keyboard
			res, err := s.bot.Send(msg)
			if err != nil {
				slog.Error(
					"error while send message with inline keyboard to telegram",
					slog.String("op", op),
					slog.String("err", err.Error()),
				)
				return 0, err
			}
			return res.MessageID, err
		}
		slog.Error(
			"got error while getting email by chatId from Repo",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return 0, err
	}

	keyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("удалить", "delete_email"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("изменить", "set_or_update_email"),
		),
	)
	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Ваш email: %s", email))
	msg.ReplyMarkup = keyboard
	res, err := s.bot.Send(msg)
	if err != nil {
		slog.Error(
			"error while send message with inline keyboard to telegram",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return 0, err
	}
	return res.MessageID, nil
}

func (s *TgBotService) SetEmail(chatId int64, email string) error {
	op := "TgBotService.SetEmail"
	err := s.repo.UpsertEmail(chatId, email)
	if err != nil {
		slog.Error(
			"got error from Repo while upserting email",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}
	return nil
}

func (s *TgBotService) DeleteEmail(chatId int64) error {
	op := "TgBotService.DeleteEmail"
	err := s.repo.DeleteEmailByChatId(chatId)
	if err != nil {
		slog.Error(
			"got error from Repo while deleting email",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return err
	}
	return nil
}
