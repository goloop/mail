package mail

import "errors"

var (
	// ErrNoSender is returned when a message has no From address and the
	// transport has no default From.
	ErrNoSender = errors.New("mail: no sender address")

	// ErrNoRecipients is returned when a message has no To/Cc/Bcc recipients.
	ErrNoRecipients = errors.New("mail: no recipients")

	// ErrEmptyAddress is returned when an Address has an empty email.
	ErrEmptyAddress = errors.New("mail: empty email address")

	// ErrInsecureAuth is returned when SMTP credentials would be sent over an
	// unencrypted connection (TLSNone) without AllowInsecureAuth set.
	ErrInsecureAuth = errors.New(
		"mail: refusing to send credentials over an unencrypted connection; " +
			"use TLS or set AllowInsecureAuth")
)
