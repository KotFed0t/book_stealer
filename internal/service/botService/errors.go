package botService

import "errors"

var (
	ErrEmailNotFound = errors.New("email not found")
	ErrBooksAreEmpty = errors.New("books are empty")
)
