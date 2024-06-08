package mail

import (
	"book_stealer_tgbot/config"
	"github.com/wneessen/go-mail"
	"log/slog"
	"time"
)

func SendFile(cfg *config.Config, filePath string, to string) error {
	m := mail.NewMsg()
	if err := m.From(cfg.Mail.Address); err != nil {
		slog.Error(
			"failed to set From address",
			slog.String("from", cfg.Mail.Address),
			slog.String("err", err.Error()),
		)
		return err
	}
	if err := m.To(to); err != nil {
		slog.Error(
			"failed to set To address",
			slog.String("to", to),
			slog.String("err", err.Error()),
		)
		return err
	}
	m.Subject("")
	m.SetBodyString(mail.TypeTextPlain, "")
	m.AttachFile(filePath)

	c, err := mail.NewClient(
		cfg.Mail.Host,
		mail.WithPort(cfg.Mail.Port),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(cfg.Mail.Address),
		mail.WithPassword(cfg.Mail.Password),
		mail.WithTimeout(120*time.Second),
	)

	if err != nil {
		slog.Error(
			"failed to create mail client",
			slog.String("to", to),
			slog.String("err", err.Error()),
		)
		return err
	}

	if err = c.DialAndSend(m); err != nil {
		slog.Error(
			"failed to send mail",
			slog.String("to", to),
			slog.String("filePath", filePath),
			slog.String("err", err.Error()),
		)
		return err
	}

	return nil
}
