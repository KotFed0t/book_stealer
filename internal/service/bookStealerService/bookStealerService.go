package bookStealerService

import (
	"book_stealer_tgbot/config"
	"context"
	"io"
)

type Cache interface {
}

type BooksParser interface {
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
