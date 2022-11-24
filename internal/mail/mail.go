package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"

	"github.com/sschwartz96/syncapod-backend/internal/config"
	"go.uber.org/zap"
)

// MailQueuer is strictly used to be able to testing other modules, stubbing the Mailer functionality
type MailQueuer interface {
	Queue(to, subject, body string)
}

// Mailer implements MailQueuer
type Mailer struct {
	logger       *zap.Logger
	smtpAddress  string
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	queueChan    chan mail
}

type mail struct {
	to      string
	subject string
	body    string
}

// NewMailer uses config to dial tcp connection to the smtp server, tests the connection,
// and closes the connection. Afterwards stores the successful address and credentials
func NewMailer(cfg *config.Config, logger *zap.Logger) (*Mailer, error) {
	// build smtp address in form of host:port
	smtpAddress := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	mailer := &Mailer{
		logger:       logger,
		smtpAddress:  smtpAddress,
		smtpHost:     cfg.SMTPHost,
		smtpPort:     cfg.SMTPPort,
		smtpUser:     cfg.SMTPUser,
		smtpPassword: cfg.SMTPPassword,
		queueChan:    make(chan mail, 100), //TODO: if we reach the 100 mark (WOW), will need to offload to disk to prevent api from freezing
	}

	smtpClient, err := mailer.createClient()
	if err != nil {
		return nil, fmt.Errorf("NewMailer() failed to create SMTP connection: %v", err)
	}

	err = smtpClient.Quit()
	if err != nil {
		return nil, fmt.Errorf("NewMailer() failed to close SMTP connection: %v", err)
	}

	return mailer, nil
}

// Queue takes "to" email, subject, body of message and queues it up to send
func (m *Mailer) Queue(to, subject, body string) {
	m.logger.Info("queuing mail", zap.String("to", to), zap.String("subject", subject))
	m.queueChan <- mail{to: to, subject: subject, body: body}
}

// Start starts an infinite loop to consume incoming messages from the queue
func (m *Mailer) Start() error {
	var client *smtp.Client
	var err error
	from := m.smtpUser
	for {
		newMsg := <-m.queueChan
		m.logger.Info("new message received", zap.String("to", newMsg.to), zap.String("subject", newMsg.subject))
		if client == nil {
			m.logger.Info("smtp client closed, creating new connection")
			for {
				client, err = m.createClient()
				if err != nil {
					// TODO: more robust error handling
					m.logger.Error("Error connecting to SMTP server, trying again in 30 seconds", zap.Error(err))
					time.Sleep(time.Second * 30)
					continue
				}
				break
			}
		}

		err := client.Mail(from)
		if err != nil {
			m.logger.Error("Error sending MAIL command to SMTP server", zap.Error(err))
			continue
		}

		err = client.Rcpt(newMsg.to)
		if err != nil {
			m.logger.Error("Error sending RCPT command to SMTP server", zap.Error(err))
			continue
		}

		w, err := client.Data()
		if err != nil {
			m.logger.Error("Error sending DATA command to SMTP server", zap.Error(err))
			continue
		}
		_, err = w.Write(createMessageData(from, newMsg.to, newMsg.subject, newMsg.body))
		if err != nil {
			m.logger.Error("Error writing message data", zap.Error(err))
			continue
		}
		err = w.Close()
		if err != nil {
			m.logger.Error("Error closing WriteCloser", zap.Error(err))
			continue
		}
		m.logger.Info("successfully sent email", zap.String("to", newMsg.to))
		if len(m.queueChan) == 0 {
			m.logger.Info("no more messages in queue closing smtp connection")
			err = client.Quit()
			if err != nil {
				m.logger.Error("Error sending QUIT to SMTP server", zap.Error(err))
				continue
			}
			client = nil
		}
	}
}

func (m *Mailer) createClient() (*smtp.Client, error) {
	// first explicitly dial tls tcp connection
	tlsConn, err := tls.Dial("tcp", m.smtpAddress, nil) // pass nil to use default root set
	if err != nil {
		return nil, fmt.Errorf("createClient() error dialing tls connection: %w", err)
	}

	// set new client using existing tls connection and hostname
	client, err := smtp.NewClient(tlsConn, m.smtpHost)
	if err != nil {
		return nil, fmt.Errorf("createClient() error creating new client: %w", err)
	}

	// set up smtp authentication and authenticate
	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPassword, m.smtpHost)
	err = client.Auth(auth)
	if err != nil {
		return nil, fmt.Errorf("createClient() failed to log into smtp server: %v", err)
	}

	err = client.Noop()
	if err != nil {
		return nil, fmt.Errorf("createClient() failed to noop smtp server: %v", err)
	}
	return client, nil
}

func createMessageData(from, to, subject, body string) []byte {
	return []byte(fmt.Sprintf("From: Syncapod <%s>\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", from, to, subject, body))
}
