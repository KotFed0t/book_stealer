package bookStealerService

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/repository"
	"book_stealer_tgbot/internal/service"
	"book_stealer_tgbot/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

//go:generate mockgen -destination=mocks/cache.go -package=mocks . Cache
type Cache interface {
	GetBooksForPage(ctx context.Context, title, author string, page int) (booksPage model.BooksPage, err error)
	SetBooksForPage(ctx context.Context, booksPage model.BooksPage) error
}

//go:generate mockgen -destination=mocks/booksParser.go -package=mocks . BooksParser
type BooksParser interface {
	GetBooksPaginated(ctx context.Context, bookTitle string, author string, limit, offset int) (books []model.BookPreview, hasNextPage bool, err error)
	ParseBookPage(ctx context.Context, ref string) (book model.Book, err error)
}

//go:generate mockgen -destination=mocks/repository.go -package=mocks . Repository
type Repository interface {
	GetEmailByChatId(ctx context.Context, chatId int64) (email string, err error)
	UpsertEmail(ctx context.Context, chatId int64, email string) error
	DeleteEmailByChatId(ctx context.Context, chatId int64) error
}

//go:generate mockgen -destination=mocks/cloudStorageApi.go -package=mocks . CloudStorageApi
type CloudStorageApi interface {
	UploadFile(ctx context.Context, reader io.Reader, filename string) (downloadLink string, err error)
}

//go:generate mockgen -destination=mocks/fileDownloader.go -package=mocks . FileDownloader
type FileDownloader interface {
	Download(ctx context.Context, url string) (fileBytes []byte, filename string, err error)
}

//go:generate mockgen -destination=mocks/mailer.go -package=mocks . Mailer
type Mailer interface {
	SendFile(ctx context.Context, to string, fileName string, fileContent []byte) error
}

type BookStealerService struct {
	cfg             *config.Config
	repo            Repository
	cache           Cache
	booksParser     BooksParser
	cloudStorageApi CloudStorageApi
	fileDownloader  FileDownloader
	mailer          Mailer
}

func New(
	cfg *config.Config,
	repo Repository,
	cache Cache,
	booksParser BooksParser,
	cloudStorageApi CloudStorageApi,
	fileDownloader FileDownloader,
	mailer Mailer,
) *BookStealerService {
	return &BookStealerService{
		cfg:             cfg,
		repo:            repo,
		cache:           cache,
		booksParser:     booksParser,
		cloudStorageApi: cloudStorageApi,
		fileDownloader:  fileDownloader,
		mailer:          mailer,
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
	return s.booksParser.ParseBookPage(ctx, bookLink)
}

func (s *BookStealerService) DownloadBook(ctx context.Context, url string) (fileBytes []byte, filename string, err error) {
	return s.fileDownloader.Download(ctx, url)
}

func (s *BookStealerService) UploadFileToCloud(ctx context.Context, reader io.Reader, filename string) (downloadLink string, err error) {
	return s.cloudStorageApi.UploadFile(ctx, reader, filename)
}

func (s *BookStealerService) GetEmail(ctx context.Context, chatID int64) (email string, err error) {
	email, err = s.repo.GetEmailByChatId(ctx, chatID)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return "", service.ErrNotFound
		}
		return "", err
	}
	return email, nil
}

func (s *BookStealerService) SetEmail(ctx context.Context, chatID int64, email string) error {
	return s.repo.UpsertEmail(ctx, chatID, email)
}

func (s *BookStealerService) DeleteEmail(ctx context.Context, chatID int64) error {
	return s.repo.DeleteEmailByChatId(ctx, chatID)
}

func (s *BookStealerService) SendBookToKindle(ctx context.Context, bookUrl string, chatID int64) error {
	email, err := s.repo.GetEmailByChatId(ctx, chatID)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return service.ErrNotFound
		}
		return fmt.Errorf("get email error: %w", err)
	}

	fileBytes, fileName, err := s.fileDownloader.Download(ctx, bookUrl)
	if err != nil {
		return fmt.Errorf("dowloand book error: %w", err)
	}

	err = s.mailer.SendFile(ctx, email, fileName, fileBytes)
	if err != nil {
		return fmt.Errorf("send file error: %w", err)
	}

	return nil
}
