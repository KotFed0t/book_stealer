package telebotConverter

import (
	"book_stealer_tgbot/internal/model"
	"book_stealer_tgbot/internal/model/tg/tgCallback.go"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"
)

func EnteredTitleMenuResponse(title string) (text string, markup *tele.ReplyMarkup) {
	markup = &tele.ReplyMarkup{}
	text = fmt.Sprintf("Вы ввели название книги: %s", title)
	enterAuthorSurnameBtn := markup.Data("указать фамилию автора", tgCallback.EnterAuthorSurname)
	searchByBookTitleBtn := markup.Data("искать по названию книги", tgCallback.SearchByBookTitle)

	markup.Inline(
		markup.Row(enterAuthorSurnameBtn),
		markup.Row(searchByBookTitleBtn),
	)

	return text, markup
}

func EnterAuthorResponse() (text string) {
	return "введите фамилию автора (без имени)"
}

func BooksNotFound(title, author string) string {
	return fmt.Sprintf("не удалось найти книг по запросу: %s %s", title, author)
}

func BooksPage(booksPage model.BooksPage, booksPerPage int) (text string, markup *tele.ReplyMarkup) {
	markup = &tele.ReplyMarkup{}
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("Результаты поиска: %s %s\n\n", booksPage.Title, booksPage.Author))

	menuRows := make([]tele.Row, 0)

	for i, book := range booksPage.Books {
		if i%5 == 0 {
			menuRows = append(menuRows, make(tele.Row, 0, 5))
		}

		ordinal := (booksPage.Page * booksPerPage) + i + 1
		sb.WriteString(fmt.Sprintf("%d) %s \n\n", ordinal, book.Title))
		btn := markup.Data(strconv.Itoa(ordinal), tgCallback.ToBookDetails+book.Link)
		menuRows[len(menuRows)-1] = append(menuRows[len(menuRows)-1], btn)
	}

	paginationBtns := make([]tele.Btn, 0)
	if booksPage.Page > 0 {
		paginationBtns = append(paginationBtns, markup.Data("назад", tgCallback.ToBooksPage+strconv.Itoa((booksPage.Page-1))))
	}

	if booksPage.Page > 0 || booksPage.HasNextPage {
		paginationBtns = append(paginationBtns, markup.Data(fmt.Sprintf("стр %d", booksPage.Page+1), tgCallback.PageNumber))
	}

	if booksPage.HasNextPage {
		paginationBtns = append(paginationBtns, markup.Data("вперед", tgCallback.ToBooksPage+strconv.Itoa((booksPage.Page+1))))
	}

	menuRows = append(menuRows, markup.Row(paginationBtns...))

	markup.Inline(menuRows...)

	return sb.String(), markup
}

func BookDetails(book model.Book) (text string, markup *tele.ReplyMarkup) {
	markup = &tele.ReplyMarkup{}
	text = fmt.Sprintf("%s\n\n%s\n\n%s\n\n", book.Title, strings.Join(book.Authors, ", "), book.Annotation)

	menuRows := make([]tele.Row, 0)
	backBtn := markup.Data("назад", tgCallback.BackToBooksPage)
	menuRows = append(menuRows, markup.Row(backBtn))

	i := 0
	for name, link := range book.DownloadRefs {
		if i%5 == 0 {
			menuRows = append(menuRows, make(tele.Row, 0, 5))
		}

		btn := markup.Data(name, tgCallback.DownloadBook+link)
		menuRows[len(menuRows)-1] = append(menuRows[len(menuRows)-1], btn)
		i++
	}

	if link, ok := book.DownloadRefs["(epub)"]; ok {
		btn := markup.Data("отправить на kindle", tgCallback.SendToKindle+link)
		menuRows = append(menuRows, markup.Row(btn))
	}

	markup.Inline(menuRows...)

	return text, markup
}

func EmailNotLinkedMenu() (text string, markup *tele.ReplyMarkup) {
	markup = &tele.ReplyMarkup{}
	text = "У вас нет привязанного email"

	btn := markup.Data("привязать", tgCallback.LinkEmail)

	markup.Inline(markup.Row(btn))

	return text, markup
}

func EmailMenu(email string) (text string, markup *tele.ReplyMarkup) {
	markup = &tele.ReplyMarkup{}
	text = fmt.Sprintf("ваш email: %s", email)

	deleteBtn := markup.Data("удалить", tgCallback.DeleteEmail)
	changeBtn := markup.Data("изменить", tgCallback.LinkEmail)

	markup.Inline(
		markup.Row(deleteBtn),
		markup.Row(changeBtn),
	)

	return text, markup
}
