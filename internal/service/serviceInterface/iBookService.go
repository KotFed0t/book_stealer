package serviceInterface

import "book_stealer_tgbot/internal/model"

type IBookService interface {
	GetBooksForPage(chatId int64, chatSession *model.ChatSession, page int) (books []model.BookPreview, hasNextPage bool, err error)
}
