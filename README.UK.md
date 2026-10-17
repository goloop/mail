[![Go Reference](https://img.shields.io/badge/godoc-reference-blue)](https://pkg.go.dev/github.com/goloop/mail) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/mail/blob/v1/LICENSE) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# mail

`mail` будує й надсилає вихідну пошту. Це малий транспорт і збирач повідомлень -
не платформа розсилок і не шаблонізатор. Нуль залежностей, лише стандартна
бібліотека.

## Встановлення

```bash
go get github.com/goloop/mail
```

## Швидкий старт

```go
sender := mail.NewSMTP(mail.SMTPConfig{
	Host: "smtp.example.com",
	From: mail.Address{Name: "App", Email: "no-reply@example.com"},
})

msg := mail.Message{
	To:      []mail.Address{{Email: "user@example.com"}},
	Subject: "Ласкаво просимо",
	Text:    "Вітаємо.",
	HTML:    "<p>Вітаємо.</p>",
}

if err := sender.Send(ctx, msg); err != nil {
	return err
}
```

Для типового випадку:

```go
mail.Send(ctx, sender, "user@example.com", "Привіт", "текст")
```

## Побудова

Задайте `Text` та/або `HTML`:

- лише text -> `text/plain; charset=utf-8`
- лише HTML -> `text/html; charset=utf-8`
- обидва -> `multipart/alternative`
- з `Attachments` -> `multipart/mixed`

Адреси валідуються, display-імена кодуються за RFC 2047, текст -
quoted-printable, вкладення - base64. Стабільні `Date`, `Message-ID` і
`MIME-Version` додаються автоматично.

## Транспорти

| Конструктор | Призначення |
|-------------|-------------|
| `NewSMTP(SMTPConfig)` | prod; STARTTLS (дефолт), implicit TLS або plain |
| `NewLogger(io.Writer)` | розробка; пише побудоване повідомлення |
| `NewCapture()` | тести; записує повідомлення, `Last`/`Messages` |

Воркер чи хендлер залежить лише від інтерфейсу `Sender`, тож dev і prod
використовують той самий шлях коду.

## Межі

`mail` робить одне надсилання за виклик і повертає помилку. Retry, черги,
шаблонізація, обробка bounce і vendor API - поза межами: це шар застосунку або
його фонових задач.

## Документація

- Англійський довідник: [DOC.md](DOC.md)
- Український довідник: [DOC.UK.md](DOC.UK.md)

## Ліцензія

MIT - див. [LICENSE](LICENSE).

## Реліз v1.0.0

Скасування контексту перериває очікування SMTP greeting та авторизації. Base64-вкладення кодуються без проміжної копії, зі збереженням CRLF і довжини рядка 76 символів. Реліз v1 зберігає поточний import path і публічний API. Майбутні несумісні зміни потребуватимуть нової головної версії.
