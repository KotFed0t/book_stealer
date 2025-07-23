package parser

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type FlibustaParser struct {
	cfg *config.Config
}

func NewFlibustaParser(cfg *config.Config) *FlibustaParser {
	return &FlibustaParser{cfg: cfg}
}

func (f *FlibustaParser) getCollector() (*colly.Collector, error) {
	op := "FlibustaParser.getCollectorWithProxy"
	c := colly.NewCollector()

	if f.cfg.ProxyUrl != "" {
		err := c.SetProxy(f.cfg.ProxyUrl)
		if err != nil {
			slog.Error(
				"Failed to set proxy",
				slog.String("op", op),
				slog.String("err", err.Error()),
			)
			return nil, err
		}
	}

	return c, nil
}

func (f *FlibustaParser) GetBooksPaginated(ctx context.Context, bookTitle string, author string, limit, offset int) (books []model.BookPreview, hasNextPage bool, err error) {
	op := "FlibustaParser.FinGetBooksPaginateddBooks"
	rqID := utils.GetRequestIDFromCtx(ctx)

	if offset < 0 || limit <= 0 {
		return nil, false, errors.New("incorrect offset or limit")
	}

	// учитывается что нумерация флибусты с 0 страницы
	fromPage := offset / f.cfg.Flibusta.BooksPerPage
	toPage := (offset + limit - 1) / f.cfg.Flibusta.BooksPerPage

	fromBookIdx := offset       // включая
	toBookIdx := offset + limit // исключая

	books = make([]model.BookPreview, 0, limit)
	for curPage := fromPage; curPage <= toPage; curPage++ {
		parsedBooks, maxPage, err := f.getBooksForPage(ctx, bookTitle, author, curPage)
		if err != nil {
			return nil, false, fmt.Errorf("error while parsing books: %w", err)
		}

		if len(parsedBooks) == 0 {
			hasNextPage = false
			break
		}

		// высчитываем индексы книг на странице
		from := curPage * f.cfg.Flibusta.BooksPerPage
		to := from + len(parsedBooks)

		if from < fromBookIdx {
			from = fromBookIdx
		}

		if to > toBookIdx {
			to = toBookIdx
		}

		// приводим индексы книг на странице к индексам слайса
		from = from - (curPage * f.cfg.Flibusta.BooksPerPage)
		to = to - (curPage * f.cfg.Flibusta.BooksPerPage)

		if from < 0 || from > len(parsedBooks)-1 || from >= to || to > len(parsedBooks) || to <= 0 {
			params := map[string]any{
				"limit":          limit,
				"offset":         offset,
				"curPage":        curPage,
				"lenParsedBooks": len(parsedBooks),
				"from":           from,
				"to":             to,
			}
			slog.Error("incorrect books index calculation", slog.String("rqID", rqID), slog.String("op", op), slog.Any("params", params))
			return nil, false, errors.New("incorrect books index calculation")
		}

		books = append(books, parsedBooks[from:to]...)

		if maxPage <= curPage && to == len(parsedBooks) {
			hasNextPage = false
			break
		} else {
			hasNextPage = true
		}
	}

	return books, hasNextPage, nil
}

func (f *FlibustaParser) getBooksForPage(ctx context.Context, bookTitle string, author string, page int) (books []model.BookPreview, maxPage int, err error) {
	op := "FlibustaParser.getBooksForPage"
	rqID := utils.GetRequestIDFromCtx(ctx)
	c, err := f.getCollector()
	if err != nil {
		slog.Error(
			"Failed to get collector with set proxy",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("err", err.Error()),
		)
		return nil, 0, err
	}

	c.OnHTML("form", func(e *colly.HTMLElement) {
		e.DOM.Find(".genre").Remove()
		e.ForEach("div", func(_ int, div *colly.HTMLElement) {
			text := strings.TrimSpace(div.Text)
			text = strings.ReplaceAll(text, "скачать:", "")
			text = strings.ReplaceAll(text, "(читать)", "")
			text = strings.ReplaceAll(text, "(fb2) -", "")
			text = strings.ReplaceAll(text, "(epub) -", "")
			text = strings.ReplaceAll(text, "(mobi)", "")
			text = strings.ReplaceAll(text, "(скачать pdf)", "")
			text = strings.ReplaceAll(text, "(скачать djvu)", "")
			re := regexp.MustCompile(`\s+`)
			text = re.ReplaceAllString(text, " ")
			link := div.ChildAttr("a[href^='/b/']", "href")
			books = append(books, model.BookPreview{Title: text, Link: link})
		})
	})

	// поиск кол-ва страниц для данного запроса
	c.OnHTML("li.pager-last.last a", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if href == "" {
			maxPage = 0
		} else {
			re := regexp.MustCompile(`javascript:pg\((\d+)\)`)
			matches := re.FindStringSubmatch(href)
			if len(matches) == 2 {
				maxPage, _ = strconv.Atoi(matches[1])
			}
		}
	})

	c.OnRequest(func(r *colly.Request) {
		slog.Info("Visiting", slog.String("op", op), slog.String("rqID", rqID), slog.String("url", r.URL.String()))
	})

	searchURL := f.cfg.Flibusta.BaseUrl + f.cfg.Flibusta.SearchPage
	params := url.Values{}
	params.Set("ab", "ab1")
	params.Set("t", bookTitle)
	params.Set("ln", author)
	params.Set("sort", "sd2")
	params.Set("page", strconv.Itoa(page))

	fullURL := searchURL + "?" + params.Encode()

	err = c.Visit(fullURL)
	if err != nil {
		slog.Error(
			"Error while visiting url",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("url", fullURL),
			slog.String("err", err.Error()),
		)
		return nil, 0, err
	}

	return books, maxPage, nil
}

func (f *FlibustaParser) ParseBookPage(ctx context.Context, ref string) (book model.Book, err error) {
	op := "FlibustaParser.ParseBookPage"
	rqID := utils.GetRequestIDFromCtx(ctx)

	c, err := f.getCollector()
	if err != nil {
		slog.Error(
			"Failed to get collector with set proxy",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("rqID", rqID),
			slog.String("err", err.Error()),
		)
		return book, err
	}

	c.OnHTML("div:has(p.genre)", func(e *colly.HTMLElement) {
		e.DOM.Find(".genre").Remove()
		e.DOM.Find("a[href$='/read']").Remove()

		book.DownloadRefs = make(map[string]string)
		e.ForEach("a[href^='/b/']", func(i int, e *colly.HTMLElement) {
			book.DownloadRefs[e.Text] = f.cfg.Flibusta.BaseUrl + e.Attr("href")
		})

		text := strings.TrimSpace(e.Text)
		text = strings.ReplaceAll(text, "скачать:", "")
		text = strings.ReplaceAll(text, "(читать)", "")
		text = strings.ReplaceAll(text, "(fb2) -", "")
		text = strings.ReplaceAll(text, "(epub) -", "")
		text = strings.ReplaceAll(text, "(mobi)", "")
		text = strings.ReplaceAll(text, "(скачать pdf)", "")
		text = strings.ReplaceAll(text, "(скачать djvu)", "")
		re := regexp.MustCompile(`\s+`)
		book.Title = re.ReplaceAllString(text, " ")
	})

	//поиск авторов
	c.OnHTML("#main", func(e *colly.HTMLElement) {
		e.DOM.Find("#content-top").Remove()
		e.ForEach("a[href^='/a/']", func(_ int, a *colly.HTMLElement) {
			book.Authors = append(book.Authors, a.Text)
		})
	})

	//поиск текста аннотации
	c.OnHTML("h2:contains('Аннотация') + p", func(e *colly.HTMLElement) {
		book.Annotation = e.Text
	})

	c.OnRequest(func(r *colly.Request) {
		slog.Info("Visiting", slog.String("op", op), slog.String("rqID", rqID), slog.String("url", r.URL.String()))
	})

	searchURL := f.cfg.Flibusta.BaseUrl + ref
	err = c.Visit(searchURL)
	if err != nil {
		slog.Error(
			"Error while visiting url",
			slog.String("op", op),
			slog.String("rqID", rqID),
			slog.String("url", searchURL),
			slog.String("err", err.Error()),
		)
		return book, err
	}
	return book, err
}
