package downloader

import (
	"archive/zip"
	"book_stealer_tgbot/utils"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type FileDownloader struct {
	client *http.Client
}

func NewFileDownloader() *FileDownloader {
	return &FileDownloader{client: &http.Client{}}
}

func (f *FileDownloader) Download(ctx context.Context, url string) (fileBytes []byte, filename string, err error) {
	rqID := utils.GetRequestIDFromCtx(ctx)
	op := "FileDownloader.Download"
	slog.Info("Download start", slog.String("rqID", rqID), slog.String("op", op), slog.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("response status not ok")
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return nil, "", fmt.Errorf("error parsing Content-Disposition: %w", err)
	}

	filename = params["filename"]
	if filename == "" {
		return nil, "", errors.New("filename is empty")
	}

	fileBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body err: %w", err)
	}

	if filepath.Ext(filename) == ".zip" {
		fileBytes, err = f.unzip(bytes.NewReader(fileBytes), int64(len(fileBytes)))
		if err != nil {
			return nil, "", fmt.Errorf("unzip err: %w", err)
		}
		
		filename = strings.TrimSuffix(filename, ".zip")
	}

	slog.Info("Download finished", slog.String("rqID", rqID), slog.String("op", op), slog.String("url", url))

	return fileBytes, filename, err
}

func (f *FileDownloader) unzip(r io.ReaderAt, size int64) (fileBytes []byte, err error) {
	reader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}

	if len(reader.File) == 0 {
		return nil, errors.New("empty archive")
	}

	file := reader.File[0]
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	fileBytes, err = io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}
