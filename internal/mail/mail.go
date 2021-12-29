package mail

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"

	"github.com/sschwartz96/syncapod-backend/internal/config"
)

type Mailer struct {
	smtpAddress string
}

// NewMailer uses config to dial tcp connection to the smtp server, tests the connection,
// and closes the connection. Afterwards stores the successful address and credentials
func NewMailer(cfg *config.Config, tlsCfg *tls.Config) (*Mailer, error) {
	tlsCfg.ServerName = "mail.syncapod.com"
	smtpAddress := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	tlsConn, err := tls.Dial("tcp", smtpAddress, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("NewMailer() error dialing tls connection: %v", err)
	}

	client, err := smtp.NewClient(tlsConn, cfg.SMTPHost)
	if err != nil {
		return nil, err
	}

	tlsState, ok := client.TLSConnectionState()
	if !ok {
		log.Println("TLS State not ok: ", tlsState)
		err = client.StartTLS(tlsCfg)
		if err != nil {
			return nil, err
		}
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	err = client.Auth(auth)
	if err != nil {
		return nil, fmt.Errorf("NewMailer() failed to log into smtp server: %v", err)
	}

	err = client.Noop()
	if err != nil {
		return nil, fmt.Errorf("NewMailer() failed to noop smtp server: %v", err)
	}

	log.Println("Connected to SMTP server fine, closing connection for later user")

	err = client.Quit()
	if err != nil {
		return nil, fmt.Errorf("NewMailer() failed to close SMTP connection: %v", err)
	}

	return &Mailer{
		smtpAddress: smtpAddress,
	}, nil
}
