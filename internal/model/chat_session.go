package model

type action int

const (
	DefaultAction action = iota
	ExpectingAuthor
	ExpectingEmail
)

type Session struct {
	Action            action
	BookTitle, Author string
	LastMsgId         int
	DownloadLinkEpub  string
	Books             []BookPreview
	MaxSitePage       int
	CurSitePage       int
	CurTgPage         int
}
