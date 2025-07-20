package model

type Book struct {
	Title        string
	Annotation   string
	Authors      []string
	DownloadRefs map[string]string
}

type BookPreview struct {
	Title string
	Link  string
}

type BooksPage struct {
	Books       []BookPreview
	HasNextPage bool
	Page        int
	Title       string
	Author      string
}
