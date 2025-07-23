package parser

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type flibustaParserSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	cfg      *config.Config
	parser   *FlibustaParser
}

func TestFlibustaParserSuite(t *testing.T) {
	suite.Run(t, new(flibustaParserSuite))
}

func (s *flibustaParserSuite) SetupSuite() {
	s.cfg = &config.Config{
		Flibusta: config.Flibusta{
			BaseUrl:      "https://test.com",
			BooksPerPage: 5,
		},
	}
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *flibustaParserSuite) SetupTest() {
	s.parser = NewFlibustaParser(s.cfg)
}

func (s *flibustaParserSuite) Test_ParseBookPage_Success() {
	defer gock.Off()

	bookRef := "/b/12/"
	book := model.Book{
		Title:      "Бойцовский клуб [litres] 709K, 168 с. ",
		Annotation: "Это — самая потрясающая и самая скандальная книга 1990-х.\n    Книга, в которой устами Чака Паланика заговорило не просто «поколение икс», но — «поколение икс» уже озлобленное, уже растерявшее свои последние иллюзии.\n    Вы смотрели фильм «Бойцовский клуб»?\n    Тогда — читайте книгу, по которой он был снят!",
		Authors: []string{
			"Чак Паланик",
			"Илья Валерьевич Кормильцев",
		},
		DownloadRefs: map[string]string{
			"(epub)": s.cfg.Flibusta.BaseUrl + "/b/522338/epub",
			"(fb2)":  s.cfg.Flibusta.BaseUrl + "/b/522338/fb2",
			"(mobi)": s.cfg.Flibusta.BaseUrl + "/b/522338/mobi",
		},
	}

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get(bookRef).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookPageSuccessResponse)

	res, err := s.parser.ParseBookPage(context.Background(), bookRef)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), book, res)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_ParseBookPage_PageNotFoundErr() {
	defer gock.Off()

	bookRef := "/b/12/"
	expectedErr := errors.New("Not Found")

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get(bookRef).
		Reply(404).
		SetHeader("Content-Type", "text/html; charset=utf-8")

	_, err := s.parser.ParseBookPage(context.Background(), bookRef)

	assert.Equal(s.T(), expectedErr, err)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_ParseBookPage_EmptyResponseBody() {
	defer gock.Off()

	bookRef := "/b/12/"
	book := model.Book{}

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get(bookRef).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString("")

	res, err := s.parser.ParseBookPage(context.Background(), bookRef)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), book, res)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_GetBooksPaginated_Success() {
	defer gock.Off()

	title := "title"
	author := "author"
	limit := 2
	offset := 2
	params := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "0",
		"sort": "sd2",
		"t":    "title",
	}
	books := []model.BookPreview{
		{
			Title: "- Сказки 7260K (fb2) - (epub) - - Александр Сернеевич Пушкин - Владимир Михайлович Конашевич",
			Link:  "/b/832560",
		},
		{
			Title: "- Сказки народов Поволжья 3844K (fb2) - (epub) - - Наталия Константиновна Нестерова - Автор Неизвестен -- Народные сказки",
			Link:  "/b/832552",
		},
	}
	hasNextPage := true

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListPage0)

	booksRes, hasNextPageRes, err := s.parser.GetBooksPaginated(context.Background(), title, author, limit, offset)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), books, booksRes)
	assert.Equal(s.T(), hasNextPage, hasNextPageRes)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_GetBooksPaginated_SuccessTwoPages() {
	defer gock.Off()

	title := "title"
	author := "author"
	limit := 2
	offset := 4
	params1 := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "0",
		"sort": "sd2",
		"t":    "title",
	}
	params2 := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "1",
		"sort": "sd2",
		"t":    "title",
	}
	books := []model.BookPreview{
		{
			Title: "- Сказки (пер. Анна Васильевна Ганзен ,Анатолий Васильевич Старостин ,Александра Ильинична Кобецкая ,Е. Игнатьева ,Юлиана Яковлевна Яхнина , ...) 4130K (fb2) - (epub) - - Ганс Христиан Андерсен - Ян Марцин Шанцер (иллюстратор)",
			Link:  "/b/832551",
		},
		{
			Title: "- Игрушечные сказки [худ. А. Соборова] 720K - Александр Александрович Фёдоров-Давыдов - Александра Сергеевна Соборова (иллюстратор)",
			Link:  "/b/826039",
		},
	}
	hasNextPage := true

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params1).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListPage0)

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params2).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListPage1)

	booksRes, hasNextPageRes, err := s.parser.GetBooksPaginated(context.Background(), title, author, limit, offset)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), books, booksRes)
	assert.Equal(s.T(), hasNextPage, hasNextPageRes)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_GetBooksPaginated_SuccessThreePages() {
	defer gock.Off()

	title := "title"
	author := "author"
	limit := 7
	offset := 4
	params1 := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "0",
		"sort": "sd2",
		"t":    "title",
	}
	params2 := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "1",
		"sort": "sd2",
		"t":    "title",
	}
	params3 := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "2",
		"sort": "sd2",
		"t":    "title",
	}
	books := []model.BookPreview{
		{
			Title: "- Сказки (пер. Анна Васильевна Ганзен ,Анатолий Васильевич Старостин ,Александра Ильинична Кобецкая ,Е. Игнатьева ,Юлиана Яковлевна Яхнина , ...) 4130K (fb2) - (epub) - - Ганс Христиан Андерсен - Ян Марцин Шанцер (иллюстратор)",
			Link:  "/b/832551",
		},
		{
			Title: "- Игрушечные сказки [худ. А. Соборова] 720K - Александр Александрович Фёдоров-Давыдов - Александра Сергеевна Соборова (иллюстратор)",
			Link:  "/b/826039",
		},
		{
			Title: "- Сказки Торгензарда. Сердце и пыль 17708K (fb2) - (epub) - - Артем Вадимович Журавлев",
			Link:  "/b/825912",
		},
		{
			Title: "- Сказки старухи-говорухи о животных (из народных сказок) [худ. С. Дудин, Н. Ткаченко] 19167K - Автор Неизвестен -- Народные сказки - Александр Александрович Фёдоров-Давыдов - Самуил Мартынович Дудин (иллюстратор, этнограф) - Николай И. Ткаченко (иллюстратор)",
			Link:  "/b/825852",
		},
		{
			Title: "- Сказки Кота-Баюна [худ. А. Комаров и др.] 17154K - Александр Александрович Фёдоров-Давыдов - Алексей Никанорович Комаров (иллюстратор) - К. Спасский (иллюстратор) - А. А. Кучеренко (иллюстратор) - Иван Евграфович Полушкин (иллюстратор)",
			Link:  "/b/825839",
		},
		{
			Title: "- Русские народные сказки [худ. А. Апсит] 144206K - Александр Александрович Фёдоров-Давыдов - Автор Неизвестен -- Народные сказки - Александр Петрович Апсит (иллюстратор)",
			Link:  "/b/825831",
		},
		{
			Title: "- Сказки [2014] [худ. А. Гардян] 43937K - Сергей Григорьевич Козлов - Анаит Р. Гардян (иллюстратор)",
			Link:  "/b/821889",
		},
	}
	hasNextPage := true

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params1).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListPage0)

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params2).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListPage1)

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params3).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListPage2)

	booksRes, hasNextPageRes, err := s.parser.GetBooksPaginated(context.Background(), title, author, limit, offset)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), books, booksRes)
	assert.Equal(s.T(), hasNextPage, hasNextPageRes)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_GetBooksPaginated_PageNotFound() {
	defer gock.Off()

	title := "title"
	author := "author"
	limit := 2
	offset := 50
	params := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "10",
		"sort": "sd2",
		"t":    "title",
	}
	books := []model.BookPreview{}
	hasNextPage := false

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString("")

	booksRes, hasNextPageRes, err := s.parser.GetBooksPaginated(context.Background(), title, author, limit, offset)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), books, booksRes)
	assert.Equal(s.T(), hasNextPage, hasNextPageRes)
	assert.Equal(s.T(), true, gock.IsDone())
}

func (s *flibustaParserSuite) Test_GetBooksPaginated_NotEnoughBooks() {
	defer gock.Off()

	title := "title"
	author := "author"
	limit := 6
	offset := 50
	params := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "10",
		"sort": "sd2",
		"t":    "title",
	}
	books := []model.BookPreview{
		{
			Title: "- Зимние сказки [2015] [худ. Н. Шароватова] 772K - Сергей Григорьевич Козлов - Наталья Шароватова (иллюстратор)",
			Link: "/b/821815",
		},
		{
			Title: "- Зимние сказки [2012] [худ. К. Павлова] 7016K - Сергей Григорьевич Козлов - Ксения Андреевна Павлова (иллюстратор)",
			Link: "/b/821813",
		},
	}
	hasNextPage := false

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params).
		Reply(200).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString(bookListLastPageWithTwoBooks)

	booksRes, hasNextPageRes, err := s.parser.GetBooksPaginated(context.Background(), title, author, limit, offset)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), books, booksRes)
	assert.Equal(s.T(), hasNextPage, hasNextPageRes)
	assert.Equal(s.T(), true, gock.IsDone())
}


func (s *flibustaParserSuite) Test_GetBooksPaginated_ParsingErr() {
	defer gock.Off()

	title := "title"
	author := "author"
	limit := 3
	offset := 0
	params := map[string]string{
		"ab":   "ab",
		"ln":   "author",
		"page": "0",
		"sort": "sd2",
		"t":    "title",
	}
	expectedErr := fmt.Errorf("error while parsing books: %w", errors.New("Bad Gateway"))

	gock.New(s.cfg.Flibusta.BaseUrl).
		Get("").
		MatchParams(params).
		Reply(502).
		SetHeader("Content-Type", "text/html; charset=utf-8").
		BodyString("")

	_, _, err := s.parser.GetBooksPaginated(context.Background(), title, author, limit, offset)

	assert.Equal(s.T(), expectedErr, err)
	assert.Equal(s.T(), true, gock.IsDone())
}