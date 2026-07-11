package mail_test

import (
	"context"
	"fmt"
	"os"

	"github.com/goloop/mail"
)

// ExampleNewSMTP configures an SMTP sender and sends a message.
func ExampleNewSMTP() {
	sender := mail.NewSMTP(mail.SMTPConfig{
		Host: "smtp.example.com",
		From: mail.Address{Name: "App", Email: "no-reply@example.com"},
	})

	msg := mail.Message{
		To:      []mail.Address{{Email: "user@example.com"}},
		Subject: "Welcome",
		Text:    "Hello and welcome.",
		HTML:    "<p>Hello and welcome.</p>",
	}

	// In a real program: err := sender.Send(context.Background(), msg)
	_ = sender
	_ = msg
	fmt.Println("configured")
	// Output: configured
}

// ExampleNewLogger prints the built message instead of sending it, useful in
// local development.
func ExampleNewLogger() {
	sender := mail.NewLogger(os.Stdout)
	_ = mail.Send(context.Background(), sender,
		"user@example.com", "Hi", "body")
	// Output is the raw MIME message (omitted here).
}
