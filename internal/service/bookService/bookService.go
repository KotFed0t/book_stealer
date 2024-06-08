package bookService

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/service/serviceInterface"
	"book_stealer_tgbot/internal/sessions"
	"log/slog"
)

type BookService struct {
	cfg      *config.Config
	session  sessions.ISession
	scrapper serviceInterface.IScrapperService
}

func NewBookService(cfg *config.Config, session sessions.ISession, scrapper serviceInterface.IScrapperService) *BookService {
	return &BookService{cfg: cfg, session: session, scrapper: scrapper}
}

func (s *BookService) GetBooksForPage(
	chatId int64,
	chatSession *model.ChatSession,
	page int,
) (
	books []model.BookPreview,
	hasNextPage bool,
	err error,
) {
	op := "bookService.GetBookList"

	if chatSession == nil {
		return nil, false, ErrNilChatSession
	}

	if lenBooks := len(chatSession.Books); lenBooks > 0 {
		books, hasNextPage, err = s.getBooksFromCacheOrDownloadMore(chatId, chatSession, page)
		if err != nil {
			slog.Error(
				"got error from getBooksFromCacheOrDownloadMore",
				slog.String("err", err.Error()),
				slog.String("op", op),
				slog.Any("chatSession", chatSession),
			)
			return nil, false, err
		}
		return books, hasNextPage, nil
	}

	// если в кэше книг нет - то это обязана быть первая страница
	if page != 1 {
		return nil, false, ErrIncorrectPage
	}

	books, maxPage, err := s.scrapper.GetBooksPaginated(chatSession.BookTitle, chatSession.Author, 0)
	if err != nil {
		slog.Error(
			"got error from scrapper.GetBooksPaginated:",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return nil, false, err
	}

	slog.Info("got books paginated")

	if len(books) == 0 {
		return nil, false, ErrBooksNotFound
	}

	chatSession.Books = books
	chatSession.MaxSitePage = maxPage
	chatSession.CurSitePage = 0
	err = s.session.SetOrUpdateChatSession(chatId, *chatSession)
	if err != nil {
		return nil, false, err
	}

	to := page * s.cfg.BooksPerPage
	hasNextPage = s.hasNextPage(len(books), to, 0, maxPage)
	if to <= len(books) {
		return books[:to], hasNextPage, nil
	}
	return books, hasNextPage, nil
}

func (s *BookService) getBooksFromCacheOrDownloadMore(
	chatId int64,
	chatSession *model.ChatSession,
	page int,
) (
	books []model.BookPreview,
	hasNextPage bool,
	err error,
) {
	// op := "bookService.getBooksFromCache"
	if chatSession == nil {
		return nil, false, ErrNilChatSession
	}

	from := (page - 1) * s.cfg.BooksPerPage
	to := page * s.cfg.BooksPerPage

	if lenBooks := len(chatSession.Books); lenBooks > 0 {
		switch {
		case from < 0 || to <= 0:
			return nil, false, ErrIncorrectPage
		case from < lenBooks && to <= lenBooks: // в диапазоне
			hasNextPage = s.hasNextPage(lenBooks, to, chatSession.CurSitePage, chatSession.MaxSitePage)
			return chatSession.Books[from:to], hasNextPage, nil
		case (from < lenBooks && to > lenBooks) || (from >= lenBooks && to > lenBooks): // частично в диапазоне или вне диапазона
			// проверяем есть ли еще страницы
			if chatSession.CurSitePage < chatSession.MaxSitePage {
				chatSession.CurSitePage++
				err = s.session.SetOrUpdateChatSession(chatId, *chatSession)
				if err != nil {
					return nil, false, err
				}
				books, _, err = s.scrapper.GetBooksPaginated(chatSession.BookTitle, chatSession.Author, chatSession.CurSitePage)
				if err != nil {
					return nil, false, err
				}
				if len(books) == 0 {
					//возвращаем сколько есть для частичного вхождения в диапазон, либо ничего, если мы вне диапазона
					if from < lenBooks {
						return chatSession.Books[from:lenBooks], false, nil
					}
					return nil, false, ErrBooksNotFound
				}
				chatSession.Books = append(chatSession.Books, books...)
				err = s.session.SetOrUpdateChatSession(chatId, *chatSession)
				if err != nil {
					return nil, false, err
				}
				lenBooks = len(chatSession.Books)
				// проверяем что теперь полностью в диапазоне - иначе вернем сколько есть
				if from < lenBooks && to <= lenBooks {
					hasNextPage = s.hasNextPage(lenBooks, to, chatSession.CurSitePage, chatSession.MaxSitePage)
					return chatSession.Books[from:to], hasNextPage, nil
				}
			}
			//возвращаем сколько есть для частичного вхождения в диапазон, либо ничего, если мы вне диапазона
			if from < lenBooks {
				return chatSession.Books[from:lenBooks], false, nil
			}
			return nil, false, ErrBooksNotFound
		}
	}
	return nil, false, ErrBooksNotFound
}

func (s *BookService) hasNextPage(lenBooks int, to int, curSitePage int, maxSitePage int) bool {
	if to < lenBooks || curSitePage < maxSitePage {
		return true
	}
	return false
}
