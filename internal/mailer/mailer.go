package mailer

import (
	"book_stealer_tgbot/config"
	"book_stealer_tgbot/utils"
	"context"
	"fmt"
	"io"
	"log/slog"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	cfg    *config.Config
	dialer *gomail.Dialer
}

func NewMailer(cfg *config.Config) *Mailer {
	dialer := gomail.NewDialer(cfg.Mail.Host, cfg.Mail.Port, cfg.Mail.Login, cfg.Mail.Password)
	return &Mailer{
		cfg:    cfg,
		dialer: dialer,
	}
}

func (m *Mailer) SendFile(ctx context.Context, to string, fileName string, fileContent []byte) error {
	rqID := utils.GetRequestIDFromCtx(ctx)
	op := "Mailer.SendFile"
	slog.Info("SendFile start", slog.String("rqID", rqID), slog.String("op", op), slog.String("to", to), slog.String("fileName", fileName))

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.cfg.Mail.Address)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", "")
	msg.SetBody("text/plain", "")
	msg.Attach(fileName, gomail.SetCopyFunc(func(w io.Writer) error {
		n, err := w.Write(fileContent)
		if err != nil {
			return err
		}

		if n != len(fileContent) {
			return io.ErrShortWrite
		}
		return nil
	}))

	err := m.dialer.DialAndSend(msg)
	if err != nil {
		return fmt.Errorf("error while dialing smtp: %w", err)
	}

	slog.Info("SendFile finished", slog.String("rqID", rqID), slog.String("op", op), slog.String("to", to), slog.String("fileName", fileName))

	return nil
}
