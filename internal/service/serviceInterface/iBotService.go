package serviceInterface

import "book_stealer_tgbot/internal/model"

type IBotService interface {
	SendMessage(chatId int64, msg string, msgId ...int) error
	SendBooksForPage(chatId int64, books []model.BookPreview, chatSession *model.ChatSession, page int, hasNextPage bool, msgId ...int) (int, error)
	SendKeyboardForTitle(chatId int64, bookTitle string, msgId ...int) (int, error)
	SendKeyboardForAuthor(chatId int64, author string, msgId int) error
	SendKeyboardForBook(chatId int64, book model.Book, msgId int) error
	SendFile(chatId int64, filePath string) error
	SendToKindle(chatId int64, downloadLink string) error
	SendKeyboardForEmailCommand(chatId int64) (int, error)
	SetEmail(chatId int64, email string) error
	DeleteEmail(chatId int64) error
}
