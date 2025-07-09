package model

type Session struct {
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
