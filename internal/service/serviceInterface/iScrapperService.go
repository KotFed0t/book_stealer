package serviceInterface

import "book_stealer_tgbot/internal/model"

type IScrapperService interface {
	GetBooksPaginated(bookTitle, author string, page int) (books []model.BookPreview, maxPage int, err error)
	ParseBookPage(ref string) (book model.Book, err error)
}
