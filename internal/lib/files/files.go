package files

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DownloadFile функция загружает файл c fileUrl в директорию dir.
// Если при запросе не нужен прокси - то параметр proxyUrl можно передать пустой строкой.
func DownloadFile(dir, fileUrl, proxyUrl string) (filePath string, err error) {
	httpClient := &http.Client{}
	if proxyUrl != "" {
		proxyURL, err := url.Parse(proxyUrl)
		if err != nil {
			return "", err
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	resp, err := httpClient.Get(fileUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("response status not ok")
	}

	// Получение имени файла из заголовка Content-Disposition
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition == "" {
		return "", errors.New("content disposition is empty")
	}

	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return "", err
	}

	filename := params["filename"]
	if filename == "" {
		return "", errors.New("filename is empty")
	}

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}

	filePath, err = CreateFile(dir, filename, resp.Body)
	if err != nil {
		return "", err
	}

	if filepath.Ext(filePath) == ".zip" {
		zipFile := filePath
		filePath, err = Unzip(zipFile, dir)
		if err != nil {
			return "", err
		}
		err = DeleteFile(zipFile)
		if err != nil {
			slog.Error("Delete zip file failed", slog.String("filename", zipFile), slog.String("error", err.Error()))
		}
	}

	return filePath, nil
}

// generateUniqueFilename проверяет существование файла и генерирует уникальное имя
func generateUniqueFilename(filename string) string {
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	i := 1

	for {
		if _, err := os.Stat(filename); errors.Is(err, fs.ErrNotExist) {
			break
		}
		filename = fmt.Sprintf("%s(%d)%s", base, i, ext)
		i++
	}

	return filename
}

func Unzip(zipPath string, dir string) (filePath string, err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	if len(r.File) == 0 {
		return "", errors.New("file is empty")
	}

	f := r.File[0] // Берем 1 файл из архива

	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	filePath, err = CreateFile(dir, f.Name, rc)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func DeleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return err
	}
	return nil
}

// CreateFile создает файл в директории dir с названием файла filename и c содержимым файла content.
// Если файл с таким именем уже существует - название будет дополнено цифровым индексом
func CreateFile(dir string, filename string, content io.Reader) (filePath string, err error) {
	filename = generateUniqueFilename(filename)
	filePath = path.Join(dir, filename)
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(outFile, content)
	if err != nil {
		slog.Error("Write file failed", slog.String("filename", filename), slog.String("err", err.Error()))
		_ = outFile.Close()
		errDelete := DeleteFile(filePath)
		if errDelete != nil {
			slog.Error("failed on delete file", slog.String("filePath", filePath), slog.String("err", errDelete.Error()))
		}
		return "", err
	}

	_ = outFile.Close()
	return filePath, nil
}
