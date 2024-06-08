package sessions

import "book_stealer_tgbot/internal/model"

type ISession interface {
	GetChatSession(chatId int64) (model.ChatSession, error)
	SetOrUpdateChatSession(chatId int64, chatSession model.ChatSession) error
	DeleteChatSession(chatId int64) error
}
