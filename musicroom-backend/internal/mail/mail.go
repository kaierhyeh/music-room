package mail

import (
	"fmt"
	"log"
	"net/smtp"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func New(c SMTPConfig) Mailer {
	if c.Host == "" {
		log.Print("mail: SMTP not configured -> dev mode (emails logged to stdout)")
		return &logMailer{}
	}
	return &smtpMailer{c: c}
}

type smtpMailer struct{ c SMTPConfig }

func (m *smtpMailer) Send(to, subject, body string) error {
	addr := m.c.Host + ":" + m.c.Port
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		m.c.From, to, subject, body,
	))
	var auth smtp.Auth
	if m.c.User != "" {
		auth = smtp.PlainAuth("", m.c.User, m.c.Password, m.c.Host)
	}
	return smtp.SendMail(addr, auth, m.c.From, []string{to}, msg)
}

type logMailer struct{}

func (m *logMailer) Send(to, subject, body string) error {
	log.Printf("mail[dev] -> to=%s subject=%q\n%s", to, subject, body)
	return nil
}
