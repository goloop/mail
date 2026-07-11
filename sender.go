package mail

import "context"

// Sender delivers a message. Implementations: SMTP (NewSMTP), a dev logger
// (NewLogger) and a test capture (NewCapture).
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Send is a convenience helper that sends a plain-text message to one
// recipient through s.
func Send(ctx context.Context, s Sender, to, subject, text string) error {
	return s.Send(ctx, Message{
		To:      []Address{{Email: to}},
		Subject: subject,
		Text:    text,
	})
}
