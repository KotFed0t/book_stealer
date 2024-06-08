package scrapperService

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/internal/model"
	"github.com/gocolly/colly/v2"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type FlibustaScrapperService struct {
	cfg *config.Config
}

func NewFlibustaScrapperService(cfg *config.Config) *FlibustaScrapperService {
	return &FlibustaScrapperService{cfg: cfg}
}

func (f *FlibustaScrapperService) getCollectorWithProxy() (*colly.Collector, error) {
	op := "FlibustaScrapperService.getCollectorWithProxy"
	c := colly.NewCollector()
	err := c.SetProxy(f.cfg.ProxyUrl)
	if err != nil {
		slog.Error(
			"Failed to set proxy",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return nil, err
	}
	return c, nil
}

func (f *FlibustaScrapperService) GetBooksPaginated(bookTitle string, author string, page int) (books []model.BookPreview, maxPage int, err error) {
	op := "FlibustaScrapperService.GetBooksPaginated"
	c, err := f.getCollectorWithProxy()
	if err != nil {
		slog.Error(
			"Failed to get collector with set proxy",
			slog.String("op", op),
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
		slog.Info("Visiting", slog.String("url", r.URL.String()))
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
			slog.String("url", fullURL),
			slog.String("err", err.Error()),
		)
		return nil, 0, err
	}

	return books, maxPage, nil
}

func (f *FlibustaScrapperService) ParseBookPage(ref string) (book model.Book, err error) {
	op := "FlibustaScrapperService.ParseBookPage"
	c, err := f.getCollectorWithProxy()
	if err != nil {
		slog.Error(
			"Failed to get collector with set proxy",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		return book, err
	}

	c.OnHTML("div:has(p.genre)", func(e *colly.HTMLElement) {
		e.DOM.Find(".genre").Remove()
		e.DOM.Find("a[href$='/read']").Remove()

		book.DownloadRefs = make(map[string]string)
		e.ForEach("a[href^='/b/']", func(i int, e *colly.HTMLElement) {
			book.DownloadRefs[e.Attr("href")] = e.Text
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
		slog.Info("Visiting", slog.String("url", r.URL.String()))
	})

	searchURL := f.cfg.Flibusta.BaseUrl + ref
	err = c.Visit(searchURL)
	if err != nil {
		slog.Error(
			"Error while visiting url",
			slog.String("op", op),
			slog.String("url", searchURL),
			slog.String("err", err.Error()),
		)
		return book, err
	}
	return book, err
}
