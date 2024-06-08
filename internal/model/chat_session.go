package model

type ChatSession struct {
	ExpectingAuthor   bool
	BookTitle, Author string
	LastMsgId         int
	ExpectingEmail    bool
	DownloadLinkEpub  string
	Books             []BookPreview
	MaxSitePage       int
	CurSitePage       int
	CurTgPage         int
}
