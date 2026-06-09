package email

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

type Config struct {
	Host        string
	Port        string
	Username    string
	Password    string
	From        string
	FrontendURL string
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

func (m *Mailer) SendVerification(to, code, confirmURL string) error {
	body := fmt.Sprintf(`<p>Confirm your Nexus Forum account:</p>
<p><a href="%s">Click here to confirm your email</a></p>
<p>Or enter this code: <strong>%s</strong></p>
<p>This link and code expire in 15 minutes.</p>`, confirmURL, code)
	return m.Send(to, "Confirm your Nexus Forum account", body)
}

func (m *Mailer) SendPasswordReset(to, resetURL string) error {
	body := fmt.Sprintf(`<p>Reset your Nexus Forum password:</p>
<p><a href="%s">Click here to reset your password</a></p>
<p>This link expires in 1 hour. If you did not request a reset, ignore this email.</p>`, resetURL)
	return m.Send(to, "Reset your Nexus Forum password", body)
}

func (m *Mailer) SendNotification(to, subject, body string) error {
	frontend := strings.TrimRight(m.cfg.FrontendURL, "/")
	if frontend == "" {
		frontend = "http://localhost:5173"
	}
	html := fmt.Sprintf(`<p>%s</p><p><a href="%s">Open Nexus Forum</a></p>`, body, frontend)
	return m.Send(to, subject, html)
}

func (m *Mailer) SendScheduledPublished(to, postTitle, postURL string) error {
	body := fmt.Sprintf(`<p>Your scheduled post <strong>%s</strong> has been published.</p>
<p><a href="%s">View post</a></p>`, postTitle, postURL)
	return m.Send(to, "Your scheduled post is live", body)
}
