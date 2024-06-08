package bookService

import "errors"

var (
	ErrNilChatSession = errors.New("nil chat session")
	ErrBooksNotFound  = errors.New("books not found")
	ErrIncorrectPage  = errors.New("incorrect page")
)
