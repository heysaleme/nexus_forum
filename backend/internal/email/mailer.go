package email

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Mailer struct {
	cfg Config
}

func NewMailer(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) Enabled() bool {
	return m.cfg.Host != "" && m.cfg.Port != "" && m.cfg.From != ""
}

func (m *Mailer) Send(to, subject, body string) error {
	if !m.Enabled() {
		slog.Warn("email not configured; message skipped", "to", to, "subject", subject)
		return fmt.Errorf("email not configured: set SMTP_HOST, SMTP_PORT, SMTP_FROM")
	}
	addr := m.cfg.Host + ":" + m.cfg.Port
	from := m.cfg.From
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func (m *Mailer) SendVerification(to, code string) error {
	body := fmt.Sprintf(`<p>Your Nexus Forum verification code is:</p><h2>%s</h2><p>This code expires in 15 minutes.</p>`, code)
	return m.Send(to, "Verify your Nexus Forum account", body)
}

func (m *Mailer) SendNotification(to, subject, body string) error {
	html := fmt.Sprintf(`<p>%s</p><p><a href="#">Open Nexus Forum</a></p>`, body)
	return m.Send(to, subject, html)
}
