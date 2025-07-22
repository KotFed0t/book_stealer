package bookStealerService

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/data/cache"
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/repository"
	"book_stealer_tgbot/internal/service"
	"book_stealer_tgbot/internal/service/bookStealerService/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type bookStealerServiceSuite struct {
	suite.Suite

	mockCtrl        *gomock.Controller
	service         *BookStealerService
	cfg             *config.Config
	repo            *mocks.MockRepository
	cache           *mocks.MockCache
	cloudStorageApi *mocks.MockCloudStorageApi
	booksParser     *mocks.MockBooksParser
	fileDownloader  *mocks.MockFileDownloader
	mailer          *mocks.MockMailer
}

func TestBookStealerServiceSuite(t *testing.T) {
	suite.Run(t, new(bookStealerServiceSuite))
}

func (s *bookStealerServiceSuite) SetupSuite() {
	s.cfg = &config.Config{
		BooksPerPage: 10,
	}
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *bookStealerServiceSuite) SetupTest() {
	s.repo = mocks.NewMockRepository(s.mockCtrl)
	s.cache = mocks.NewMockCache(s.mockCtrl)
	s.cloudStorageApi = mocks.NewMockCloudStorageApi(s.mockCtrl)
	s.booksParser = mocks.NewMockBooksParser(s.mockCtrl)
	s.fileDownloader = mocks.NewMockFileDownloader(s.mockCtrl)
	s.mailer = mocks.NewMockMailer(s.mockCtrl)

	s.service = New(s.cfg, s.repo, s.cache, s.booksParser, s.cloudStorageApi, s.fileDownloader, s.mailer)
}

func (s *bookStealerServiceSuite) Test_SetEmail_Success() {
	var chatID int64 = 1
	email := "test@gmail.com"
	ctx := context.Background()

	s.repo.EXPECT().
		UpsertEmail(ctx, chatID, email).
		Return(nil)

	err := s.service.SetEmail(ctx, chatID, email)

	assert.Nil(s.T(), err)
}

func (s *bookStealerServiceSuite) Test_DeleteEmail_Success() {
	var chatID int64 = 1
	ctx := context.Background()

	s.repo.EXPECT().
		DeleteEmailByChatId(ctx, chatID).
		Return(nil)

	err := s.service.DeleteEmail(ctx, chatID)

	assert.Nil(s.T(), err)
}

func (s *bookStealerServiceSuite) Test_GetEmail_Success() {
	var chatID int64 = 1
	email := "test@gmail.com"
	ctx := context.Background()

	s.repo.EXPECT().
		GetEmailByChatId(ctx, chatID).
		Return(email, nil)

	res, err := s.service.GetEmail(ctx, chatID)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), email, res)
}

func (s *bookStealerServiceSuite) Test_GetEmail_NotFoundErr() {
	var chatID int64 = 1
	ctx := context.Background()
	expectedErr := service.ErrNotFound

	s.repo.EXPECT().
		GetEmailByChatId(ctx, chatID).
		Return("", repository.ErrNoRows)

	_, err := s.service.GetEmail(ctx, chatID)

	assert.Equal(s.T(), expectedErr, err)
}

func (s *bookStealerServiceSuite) Test_GetEmail_RepoErr() {
	var chatID int64 = 1
	ctx := context.Background()
	expectedErr := errors.New("some error")

	s.repo.EXPECT().
		GetEmailByChatId(ctx, chatID).
		Return("", expectedErr)

	_, err := s.service.GetEmail(ctx, chatID)

	assert.Equal(s.T(), expectedErr, err)
}

func (s *bookStealerServiceSuite) Test_SendBookToKindle_Success() {
	var chatID int64 = 1
	ctx := context.Background()
	email := "test@gmail.com"
	bookUrl := "bookUrl"
	fileBytes := []byte("book content")
	fileName := "fileName"

	s.repo.EXPECT().
		GetEmailByChatId(ctx, chatID).
		Return(email, nil)

	s.fileDownloader.EXPECT().
		Download(ctx, bookUrl).
		Return(fileBytes, fileName, nil)

	s.mailer.EXPECT().
		SendFile(ctx, email, fileName, fileBytes).
		Return(nil)

	err := s.service.SendBookToKindle(ctx, bookUrl, chatID)

	assert.Nil(s.T(), err)
}

func (s *bookStealerServiceSuite) Test_SendBookToKindle_EmailNotFoundErr() {
	var chatID int64 = 1
	ctx := context.Background()
	bookUrl := "bookUrl"
	expectedErr := service.ErrNotFound

	s.repo.EXPECT().
		GetEmailByChatId(ctx, chatID).
		Return("", repository.ErrNoRows)

	err := s.service.SendBookToKindle(ctx, bookUrl, chatID)

	assert.Equal(s.T(), expectedErr, err)
}

func (s *bookStealerServiceSuite) Test_SendBookToKindle_DownloadErr() {
	var chatID int64 = 1
	ctx := context.Background()
	email := "test@gmail.com"
	bookUrl := "bookUrl"
	fileDownloaderErr := errors.New("fileDownloaderErr")
	expectedErr := fmt.Errorf("dowloand book error: %w", fileDownloaderErr)

	s.repo.EXPECT().
		GetEmailByChatId(ctx, chatID).
		Return(email, nil)

	s.fileDownloader.EXPECT().
		Download(ctx, bookUrl).
		Return(nil, "", fileDownloaderErr)

	err := s.service.SendBookToKindle(ctx, bookUrl, chatID)

	assert.Equal(s.T(), expectedErr, err)
}

func (s *bookStealerServiceSuite) Test_UploadFileToCloud_Success() {
	ctx := context.Background()
	reader := bytes.NewReader([]byte("file content"))
	fileName := "fileName"
	downloadLink := "downloadLink"

	s.cloudStorageApi.EXPECT().
		UploadFile(ctx, reader, fileName).
		Return(downloadLink, nil)

	res, err := s.service.UploadFileToCloud(ctx, reader, fileName)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), downloadLink, res)
}

func (s *bookStealerServiceSuite) Test_DownloadBook_Success() {
	ctx := context.Background()
	url := "url"
	fileBytes := []byte("file content")
	fileName := "fileName"

	s.fileDownloader.EXPECT().
		Download(ctx, url).
		Return(fileBytes, fileName, nil)

	resFileBytes, resFileName, err := s.service.DownloadBook(ctx, url)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), fileBytes, resFileBytes)
	assert.Equal(s.T(), fileName, resFileName)
}

func (s *bookStealerServiceSuite) Test_GetBookDetails_Success() {
	ctx := context.Background()
	bookUrl := "bookUrl"
	book := model.Book{
		Title:      "title",
		Annotation: "Annotations",
		Authors:    []string{"author 1", "author 2"},
		DownloadRefs: map[string]string{
			"(epub)": "epubDownloadLink",
			"(fb2)":  "fb2DownloadLink",
		},
	}

	s.booksParser.EXPECT().
		ParseBookPage(ctx, bookUrl).
		Return(book, nil)

	res, err := s.service.GetBookDetails(ctx, bookUrl)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), book, res)
}

func (s *bookStealerServiceSuite) Test_GetBooksForPage_SuccessNotFromCache() {
	ctx := context.Background()
	request := model.BookSearchRequest{
		Title:  "title",
		Author: "author",
		Page:   0,
	}
	limit := 10
	offset := 0
	books := []model.BookPreview{
		{
			Title: "title1",
			Link:  "link1",
		},
		{
			Title: "title2",
			Link:  "link2",
		},
		{
			Title: "title3",
			Link:  "link3",
		},
	}
	hasNextPage := true
	booksPage := model.BooksPage{
		Books:       books,
		HasNextPage: hasNextPage,
		Page:        request.Page,
		Title:       request.Title,
		Author:      request.Author,
	}

	s.cache.EXPECT().
		GetBooksForPage(ctx, request.Title, request.Author, request.Page).
		Return(model.BooksPage{}, cache.ErrNotFound)

	s.booksParser.EXPECT().
		GetBooksPaginated(ctx, request.Title, request.Author, limit, offset).
		Return(books, hasNextPage, nil)

	s.cache.EXPECT().
		SetBooksForPage(context.WithoutCancel(ctx), booksPage).
		Return(nil)

	res, err := s.service.GetBooksForPage(ctx, request)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), booksPage, res)
}

func (s *bookStealerServiceSuite) Test_GetBooksForPage_SuccessFromCache() {
	ctx := context.Background()
	request := model.BookSearchRequest{
		Title:  "title",
		Author: "author",
		Page:   0,
	}
	books := []model.BookPreview{
		{
			Title: "title1",
			Link:  "link1",
		},
		{
			Title: "title2",
			Link:  "link2",
		},
		{
			Title: "title3",
			Link:  "link3",
		},
	}
	hasNextPage := true
	booksPage := model.BooksPage{
		Books:       books,
		HasNextPage: hasNextPage,
		Page:        request.Page,
		Title:       request.Title,
		Author:      request.Author,
	}

	s.cache.EXPECT().
		GetBooksForPage(ctx, request.Title, request.Author, request.Page).
		Return(booksPage, nil)

	res, err := s.service.GetBooksForPage(ctx, request)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), booksPage, res)
}

func (s *bookStealerServiceSuite) Test_GetBooksForPage_IncorrectPageErr() {
	ctx := context.Background()
	request := model.BookSearchRequest{
		Title:  "title",
		Author: "author",
		Page:   -1,
	}
	expectedErr := errors.New("page must be positive")

	_, err := s.service.GetBooksForPage(ctx, request)

	assert.Equal(s.T(), expectedErr, err)
}

func (s *bookStealerServiceSuite) Test_GetBooksForPage_BooksNotFoundErr() {
	ctx := context.Background()
	request := model.BookSearchRequest{
		Title:  "title",
		Author: "author",
		Page:   0,
	}
	limit := 10
	offset := 0
	var books []model.BookPreview
	hasNextPage := false
	expectedErr := service.ErrNotFound

	s.cache.EXPECT().
		GetBooksForPage(ctx, request.Title, request.Author, request.Page).
		Return(model.BooksPage{}, cache.ErrNotFound)

	s.booksParser.EXPECT().
		GetBooksPaginated(ctx, request.Title, request.Author, limit, offset).
		Return(books, hasNextPage, nil)

	_, err := s.service.GetBooksForPage(ctx, request)

	assert.Equal(s.T(), expectedErr, err)
}

func (s *bookStealerServiceSuite) Test_GetBooksForPage_ParsingErr() {
	ctx := context.Background()
	request := model.BookSearchRequest{
		Title:  "title",
		Author: "author",
		Page:   0,
	}
	limit := 10
	offset := 0
	parserErr := errors.New("parsingErr")
	expectedErr := fmt.Errorf("error while parsing books: %w", parserErr)

	s.cache.EXPECT().
		GetBooksForPage(ctx, request.Title, request.Author, request.Page).
		Return(model.BooksPage{}, cache.ErrNotFound)

	s.booksParser.EXPECT().
		GetBooksPaginated(ctx, request.Title, request.Author, limit, offset).
		Return(nil, false, parserErr)

	_, err := s.service.GetBooksForPage(ctx, request)

	assert.Equal(s.T(), expectedErr, err)
}
