package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"time"
)

// TLSMode selects how the SMTP connection is secured.
type TLSMode int

const (
	// TLSStartTLS upgrades a plain connection with STARTTLS (default).
	TLSStartTLS TLSMode = iota
	// TLSImplicit dials a TLS connection directly (SMTPS, usually port 465).
	TLSImplicit
	// TLSNone uses a plain, unencrypted connection (dev only).
	TLSNone
)

// defaultSMTPTimeout bounds the whole send unless overridden.
const defaultSMTPTimeout = 10 * time.Second

// SMTPConfig configures an SMTP Sender.
type SMTPConfig struct {
	Host     string
	Port     int // defaults to 587
	Username string
	Password string
	From     Address // default From when a message has none
	TLS      TLSMode
	Timeout  time.Duration

	// AllowInsecureAuth permits sending credentials over an unencrypted
	// connection (TLSNone). It is off by default: credentials never travel in
	// cleartext unless you opt in, which is intended only for local testing.
	AllowInsecureAuth bool
}

func (c SMTPConfig) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return defaultSMTPTimeout
}

func (c SMTPConfig) port() int {
	if c.Port > 0 {
		return c.Port
	}
	return 587
}

// smtpSender sends messages over SMTP.
type smtpSender struct {
	cfg SMTPConfig
}

// NewSMTP returns a Sender that delivers messages over SMTP, honouring the
// configured TLS mode, credentials and timeout.
func NewSMTP(cfg SMTPConfig) Sender {
	return &smtpSender{cfg: cfg}
}

// Send builds and delivers the message to its recipients.
func (s *smtpSender) Send(ctx context.Context, msg Message) error {
	if msg.From.Email == "" {
		msg.From = s.cfg.From
	}
	if msg.From.Email == "" {
		return ErrNoSender
	}
	recipients := msg.recipients()
	if len(recipients) == 0 {
		return ErrNoRecipients
	}

	raw, err := msg.build()
	if err != nil {
		return err
	}

	client, err := s.dial(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Abort the exchange if the caller cancels the context.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-done:
		}
	}()

	bcc := bccSet(msg)

	if err := client.Mail(msg.From.Email); err != nil {
		return fmt.Errorf("mail: MAIL FROM <%s>: %w", msg.From.Email, err)
	}
	for _, r := range recipients {
		if err := client.Rcpt(r); err != nil {
			// Do not put a blind recipient's address in the error: a caller
			// that logs or shows it would leak the Bcc.
			if bcc[r] {
				return fmt.Errorf("mail: RCPT TO (blind recipient): %w", err)
			}
			return fmt.Errorf("mail: RCPT TO <%s>: %w", r, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		_ = w.Close()
		return fmt.Errorf("mail: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close body: %w", err)
	}

	// The server accepted the message once DATA closed cleanly. A failure to
	// QUIT after that (context cancel, dirty close) does not un-send it, so do
	// not report it as a delivery failure and risk a duplicate on retry.
	_ = client.Quit()
	return nil
}

// bccSet returns the set of Bcc recipient emails, for masking their addresses
// in errors so a blind recipient is not disclosed.
func bccSet(msg Message) map[string]bool {
	if len(msg.Bcc) == 0 {
		return nil
	}
	set := make(map[string]bool, len(msg.Bcc))
	for _, a := range msg.Bcc {
		if a.Email != "" {
			set[a.Email] = true
		}
	}
	return set
}

// dial opens an SMTP client, applying the TLS mode, deadline and auth.
func (s *smtpSender) dial(ctx context.Context) (*smtp.Client, error) {
	// Never send credentials in cleartext by default. AUTH PLAIN over an
	// unencrypted connection exposes the password, so refuse it unless the
	// caller has explicitly opted in for local testing.
	if s.cfg.TLS == TLSNone && s.cfg.Username != "" && !s.cfg.AllowInsecureAuth {
		return nil, ErrInsecureAuth
	}

	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.port()))
	d := &net.Dialer{Timeout: s.cfg.timeout()}

	var conn net.Conn
	var err error
	if s.cfg.TLS == TLSImplicit {
		td := &tls.Dialer{NetDialer: d, Config: &tls.Config{ServerName: s.cfg.Host}}
		conn, err = td.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, err
	}

	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(s.cfg.timeout()))
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if s.cfg.TLS == TLSStartTLS {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}
