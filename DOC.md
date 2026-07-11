# mail - reference

`mail` builds RFC 5322/MIME messages and delivers them over SMTP. Full English
reference; Ukrainian in [DOC.UK.md](DOC.UK.md).

## Contents

- [Addresses](#addresses)
- [Messages](#messages)
- [MIME structure](#mime-structure)
- [Senders](#senders)
- [SMTP configuration](#smtp-configuration)
- [Context and timeouts](#context-and-timeouts)
- [Scope](#scope)

## Addresses

```go
type Address struct {
	Name  string
	Email string
}
```

`String` formats the address per RFC 5322, encoding a non-ASCII display name as
an RFC 2047 encoded-word. `Validate` reports whether the email is non-empty and
parseable. Building a message validates every address and fails on the first
invalid one.

## Messages

```go
type Message struct {
	From        Address
	To, Cc, Bcc []Address
	ReplyTo     []Address
	Subject     string
	Text        string
	HTML        string
	Attachments []Attachment
	Headers     map[string]string
}
```

`Headers` adds extra headers; structural and envelope headers (From, To, Cc,
Bcc, Reply-To, Subject, Date, Message-ID, MIME-Version, Content-*) cannot be
overridden. `Bcc` recipients receive the message but do not appear in the
headers.

## MIME structure

The body is chosen from `Text`/`HTML`:

| Content | Structure |
|---------|-----------|
| text only | `text/plain; charset=utf-8` |
| HTML only | `text/html; charset=utf-8` |
| both | `multipart/alternative` (plain then HTML) |
| any attachments | `multipart/mixed` wrapping the above |

Text parts are quoted-printable; attachments are base64 wrapped at 76 columns
with a sanitized filename. The subject is RFC 2047 encoded when it contains
non-ASCII characters. A random `Message-ID` and a stable `Date` are always set.

## Senders

```go
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
```

- `NewSMTP(SMTPConfig)` delivers over SMTP.
- `NewLogger(io.Writer)` writes the built message (development).
- `NewCapture()` records messages (tests): `Messages`, `Last`, `Reset`.

The `Send(ctx, sender, to, subject, text)` helper covers the plain-text case.

## SMTP configuration

```go
type SMTPConfig struct {
	Host     string
	Port     int     // default 587
	Username string
	Password string
	From     Address // default From when a message has none
	TLS      TLSMode
	Timeout  time.Duration // default 10s
}
```

`TLSMode` is `TLSStartTLS` (default), `TLSImplicit` (SMTPS, usually port 465) or
`TLSNone` (development only). When `Username` is set, `PLAIN` auth is used over
the secured connection.

The send sequence: validate sender/recipients, build the message, dial with the
timeout, upgrade to TLS if configured, authenticate, then `MAIL`/`RCPT`/`DATA`
and `QUIT`.

## Context and timeouts

The connection deadline is taken from the context deadline, or the configured
timeout when the context has none. Cancelling the context closes the connection
and aborts the exchange.

## Scope

`mail` does one send per call and returns the error. Retries, queues,
templating, bounce handling, inbound email and vendor APIs are out of scope.
