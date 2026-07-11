// Package mail builds and sends outbound email, standard library only.
//
// It is a small transport and message builder, not a marketing platform or a
// template engine: it constructs an RFC 5322/MIME message and delivers it over
// SMTP, with drop-in transports for development and tests.
//
// # Building
//
// Set Text and/or HTML on a Message; both yields multipart/alternative, and
// any attachments wrap the body in multipart/mixed. Addresses are validated
// and display names are RFC 2047 encoded.
//
//	msg := mail.Message{
//	    From:    mail.Address{Name: "App", Email: "no-reply@example.com"},
//	    To:      []mail.Address{{Email: "user@example.com"}},
//	    Subject: "Welcome",
//	    Text:    "Hello and welcome.",
//	    HTML:    "<p>Hello and welcome.</p>",
//	}
//
// # Sending
//
// A Sender delivers a message. NewSMTP sends over SMTP (STARTTLS by default,
// implicit TLS or plain optional); NewLogger writes the built message to a
// writer for development; NewCapture records messages for tests.
//
//	sender := mail.NewSMTP(mail.SMTPConfig{
//	    Host: "smtp.example.com",
//	    From: mail.Address{Email: "no-reply@example.com"},
//	})
//	if err := sender.Send(ctx, msg); err != nil {
//	    return err
//	}
//
// The Send helper covers the common plain-text case:
//
//	mail.Send(ctx, sender, "user@example.com", "Hi", "body")
//
// # Scope
//
// mail does one send per call and returns the error. Retries, queues,
// templating, bounce handling and vendor APIs are out of scope: they belong to
// the surrounding application or its background job layer.
//
// See DOC.md (English) and DOC.UK.md (Ukrainian) for the full reference.
package mail
