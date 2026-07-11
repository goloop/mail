[![Go Reference](https://img.shields.io/badge/godoc-reference-blue)](https://pkg.go.dev/github.com/goloop/mail) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/mail/blob/master/LICENSE) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# mail

`mail` builds and sends outbound email. It is a small transport and message
builder - not a marketing platform or a template engine. Zero dependencies,
standard library only.

## Install

```bash
go get github.com/goloop/mail
```

## Quick start

```go
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

if err := sender.Send(ctx, msg); err != nil {
	return err
}
```

For the common case:

```go
mail.Send(ctx, sender, "user@example.com", "Hi", "body")
```

## Building

Set `Text` and/or `HTML`:

- text only -> `text/plain; charset=utf-8`
- HTML only -> `text/html; charset=utf-8`
- both -> `multipart/alternative`
- with `Attachments` -> `multipart/mixed`

Addresses are validated, display names are RFC 2047 encoded, text is
quoted-printable and attachments are base64. Stable `Date`, `Message-ID` and
`MIME-Version` headers are added automatically.

## Transports

| Constructor | Use |
|-------------|-----|
| `NewSMTP(SMTPConfig)` | production; STARTTLS (default), implicit TLS, or plain |
| `NewLogger(io.Writer)` | development; writes the built message |
| `NewCapture()` | tests; records messages, inspect with `Last`/`Messages` |

A worker or handler depends only on the `Sender` interface, so dev and prod use
the same code path.

## Scope

`mail` does one send per call and returns the error. Retries, queues,
templating, bounce handling and vendor APIs are out of scope: they belong to the
surrounding application or its background job layer.

## Documentation

- English reference: [DOC.md](DOC.md)
- Ukrainian reference: [DOC.UK.md](DOC.UK.md)

## License

MIT - see [LICENSE](LICENSE).
