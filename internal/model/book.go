package model

type Book struct {
	Title        string
	Annotation   string
	Authors      []string
	DownloadRefs map[string]string
}
