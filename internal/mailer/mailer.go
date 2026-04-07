package mailer

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	mailer "github.com/xhit/go-simple-mail/v2"

	"github.com/shurco/mycart/internal/models"
)

var EncryptionTypes = map[string]mailer.Encryption{
	"None":     mailer.EncryptionNone,
	"SSL/TLS":  mailer.EncryptionSSL,
	"STARTTLS": mailer.EncryptionTLS,
}

// SendMail sends an email using the provided SMTP settings and message data.
func SendMail(smtp *models.Mail, mail *models.MessageMail) error {
	// Validate SMTP settings before attempting connection
	if smtp.SMTP.Host == "" || smtp.SMTP.Port <= 0 || smtp.SMTP.Username == "" || smtp.SMTP.Password == "" {
		return fmt.Errorf("invalid SMTP settings: host, port, username, and password are required")
	}

	// Validate sender email and name
	if smtp.SenderEmail == "" {
		return fmt.Errorf("sender email is required")
	}

	server := mailer.NewSMTPClient()
	server.Host = smtp.SMTP.Host
	server.Port = smtp.SMTP.Port
	server.Username = smtp.SMTP.Username
	server.Password = smtp.SMTP.Password
	server.Encryption = EncryptionTypes[smtp.SMTP.Encryption]

	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		return err
	}

	// Format From address: use email only if name is empty, otherwise use "Name <email>"
	var from string
	if smtp.SenderName != "" {
		from = fmt.Sprintf("%s <%s>", smtp.SenderName, smtp.SenderEmail)
	} else {
		from = smtp.SenderEmail
	}

	email := mailer.NewMSG()
	email.SetFrom(from).
		AddTo(mail.To).
		SetSubject(mail.Letter.Subject)

	bodyText, err := textTemplate(mail.Letter.Text, mail.Data)
	if err != nil {
		return err
	}
	email.SetBodyData(mailer.TextPlain, bodyText)
	// email.AddAlternativeData(mail.TextPlain, "Hello Gophers!")

	if len(mail.Files) > 0 {
		for _, file := range mail.Files {
			email.Attach(&mailer.File{
				FilePath: fmt.Sprintf("./lc_digitals/%s.%s", file.Name, file.Ext),
				Name:     file.OrigName,
			})
		}
	}

	if err := email.Send(smtpClient); err != nil {
		return err
	}

	return nil
}

func textTemplate(tmp string, data any) ([]byte, error) {
	tmpl, err := template.New("").Parse(tmp)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}
