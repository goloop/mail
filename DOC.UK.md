# mail - довідник

`mail` будує повідомлення RFC 5322/MIME і доставляє їх через SMTP. Повний
український довідник; англійською - [DOC.md](DOC.md).

## Зміст

- [Адреси](#адреси)
- [Повідомлення](#повідомлення)
- [MIME-структура](#mime-структура)
- [Sender-и](#sender-и)
- [Конфігурація SMTP](#конфігурація-smtp)
- [Контекст і таймаути](#контекст-і-таймаути)
- [Межі](#межі)

## Адреси

```go
type Address struct {
	Name  string
	Email string
}
```

`String` форматує адресу за RFC 5322, кодуючи не-ASCII display-ім'я як
encoded-word за RFC 2047. `Validate` каже, чи email непорожній і парситься.
Побудова повідомлення валідує кожну адресу й падає на першій некоректній.

## Повідомлення

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

`Headers` додає додаткові заголовки; структурні й конвертні (From, To, Cc, Bcc,
Reply-To, Subject, Date, Message-ID, MIME-Version, Content-*) перевизначити не
можна. `Bcc`-отримувачі отримують лист, але не з'являються в заголовках.

## MIME-структура

Тіло обирається з `Text`/`HTML`:

| Вміст | Структура |
|-------|-----------|
| лише text | `text/plain; charset=utf-8` |
| лише HTML | `text/html; charset=utf-8` |
| обидва | `multipart/alternative` (plain, потім HTML) |
| будь-які вкладення | `multipart/mixed` навколо вищого |

Text-частини - quoted-printable; вкладення - base64 з переносом на 76 колонок і
санітизованим ім'ям файлу. Subject кодується за RFC 2047, коли містить не-ASCII.
Випадковий `Message-ID` і стабільний `Date` встановлюються завжди.

## Sender-и

```go
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
```

- `NewSMTP(SMTPConfig)` доставляє через SMTP.
- `NewLogger(io.Writer)` пише побудоване повідомлення (розробка).
- `NewCapture()` записує повідомлення (тести): `Messages`, `Last`, `Reset`.

Хелпер `Send(ctx, sender, to, subject, text)` покриває plain-text випадок.

## Конфігурація SMTP

```go
type SMTPConfig struct {
	Host     string
	Port     int     // дефолт 587
	Username string
	Password string
	From     Address // From за замовчуванням, коли в повідомленні немає
	TLS      TLSMode
	Timeout  time.Duration // дефолт 10s
}
```

`TLSMode` - це `TLSStartTLS` (дефолт), `TLSImplicit` (SMTPS, зазвичай порт 465)
або `TLSNone` (лише розробка). Коли задано `Username`, використовується `PLAIN`-
auth поверх захищеного з'єднання.

Послідовність надсилання: валідація sender/recipients, побудова повідомлення,
dial із таймаутом, апгрейд до TLS за конфігом, автентифікація, тоді
`MAIL`/`RCPT`/`DATA` і `QUIT`.

## Контекст і таймаути

Дедлайн з'єднання береться з дедлайну контексту, або з налаштованого таймауту,
коли контекст його не має. Скасування контексту закриває з'єднання й перериває
обмін.

## Межі

`mail` робить одне надсилання за виклик і повертає помилку. Retry, черги,
шаблонізація, обробка bounce, вхідна пошта і vendor API - поза межами.
