package repository

type IRepository interface {
	GetEmailByChatId(chatId int64) (email string, err error)
	UpsertEmail(chatId int64, email string) error
	DeleteEmailByChatId(chatId int64) error
}
