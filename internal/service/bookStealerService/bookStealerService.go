package bookStealerService

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/service"
	"book_stealer_tgbot/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

type Cache interface {
	GetBooksForPage(ctx context.Context, title, author string, page int) (booksPage model.BooksPage, err error)
	SetBooksForPage(ctx context.Context, booksPage model.BooksPage) error
}

type BooksParser interface {
	GetBooksPaginated(ctx context.Context, bookTitle string, author string, limit, offset int) (books []model.BookPreview, hasNextPage bool, err error)
	ParseBookPage(ref string) (book model.Book, err error)
}

type Repository interface {
}

type CloudStorageApi interface {
	UploadFile(ctx context.Context, reader io.Reader, filename string) (downloadLink string, err error)
}

type BookStealerService struct {
	cfg             *config.Config
	repo            Repository
	cache           Cache
	booksParser     BooksParser
	cloudStorageApi CloudStorageApi
}

func New(cfg *config.Config, repo Repository, cache Cache, booksParser BooksParser, cloudStorageApi CloudStorageApi) *BookStealerService {
	return &BookStealerService{
		cfg:             cfg,
		repo:            repo,
		cache:           cache,
		booksParser:     booksParser,
		cloudStorageApi: cloudStorageApi,
	}
}

func (s *BookStealerService) GetBooksForPage(ctx context.Context, request model.BookSearchRequest) (booksPage model.BooksPage, err error) {
	op := "BookStealerService.getBooksForPage"
	rqID := utils.GetRequestIDFromCtx(ctx)

	if request.Page < 0 {
		return model.BooksPage{}, errors.New("page must be positive")
	}

	booksPage, err = s.cache.GetBooksForPage(ctx, request.Title, request.Author, request.Page)
	if err == nil {
		slog.Debug("found books page in cache", slog.String("rqID", rqID), slog.String("op", op))
		return booksPage, nil
	}

	offset := request.Page * s.cfg.BooksPerPage
	limit := s.cfg.BooksPerPage

	books, hasNextPage, err := s.booksParser.GetBooksPaginated(ctx, request.Title, request.Author, limit, offset)
	if err != nil {
		return model.BooksPage{}, fmt.Errorf("error while parsing books: %w", err)
	}

	if len(books) == 0 {
		return model.BooksPage{}, service.ErrNotFound
	}

	booksPage = model.BooksPage{
		Books:       books,
		HasNextPage: hasNextPage,
		Page:        request.Page,
		Title:       request.Title,
		Author:      request.Author,
	}

	go s.cache.SetBooksForPage(context.WithoutCancel(ctx), booksPage)

	return booksPage, nil
}

func (s *BookStealerService) GetBookDetails(ctx context.Context, bookLink string) (book model.Book, err error) {
	return s.booksParser.ParseBookPage(bookLink)
}
