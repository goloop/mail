package mail

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestAddressStringAndValidate(t *testing.T) {
	a := Address{Name: "Ада Лавлейс", Email: "ada@example.com"}
	if !strings.Contains(a.String(), "<ada@example.com>") {
		t.Fatalf("String: %q", a.String())
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("valid address rejected: %v", err)
	}
	if err := (Address{Email: "not an email"}).Validate(); err == nil {
		t.Fatal("expected invalid address error")
	}
	if err := (Address{}).Validate(); err != ErrEmptyAddress {
		t.Fatalf("expected ErrEmptyAddress, got %v", err)
	}
}

// parseBuilt parses built bytes back into a mail.Message for structural asserts.
func parseBuilt(t *testing.T, raw []byte) *mail.Message {
	t.Helper()
	m, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage: %v\n%s", err, raw)
	}
	return m
}

func TestBuildPlainText(t *testing.T) {
	msg := Message{
		From:    Address{Email: "a@example.com"},
		To:      []Address{{Email: "b@example.com"}},
		Subject: "Привіт",
		Text:    "hello",
	}
	raw, err := msg.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m := parseBuilt(t, raw)
	if ct := m.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content-type: %q", ct)
	}
	// Subject must be RFC 2047 encoded for the non-ASCII value.
	dec := new(mime.WordDecoder)
	subj, _ := dec.DecodeHeader(m.Header.Get("Subject"))
	if subj != "Привіт" {
		t.Fatalf("subject decode: %q", subj)
	}
	body, _ := io.ReadAll(m.Body)
	if !strings.Contains(string(body), "hello") {
		t.Fatalf("body: %q", body)
	}
}

func TestBuildAlternative(t *testing.T) {
	msg := Message{
		From: Address{Email: "a@example.com"},
		To:   []Address{{Email: "b@example.com"}},
		Text: "plain",
		HTML: "<p>rich</p>",
	}
	raw, err := msg.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m := parseBuilt(t, raw)
	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/alternative" {
		t.Fatalf("media type: %q err=%v", mediaType, err)
	}
	mr := multipart.NewReader(m.Body, params["boundary"])
	var types []string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		mt, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		types = append(types, mt)
	}
	if strings.Join(types, ",") != "text/plain,text/html" {
		t.Fatalf("parts: %v", types)
	}
}

func TestBuildMixedWithAttachment(t *testing.T) {
	msg := Message{
		From:        Address{Email: "a@example.com"},
		To:          []Address{{Email: "b@example.com"}},
		Text:        "see attached",
		Attachments: []Attachment{{Filename: "note.txt", Data: []byte("file body")}},
	}
	raw, err := msg.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m := parseBuilt(t, raw)
	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/mixed" {
		t.Fatalf("media type: %q err=%v", mediaType, err)
	}
	mr := multipart.NewReader(m.Body, params["boundary"])
	var foundAttachment bool
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		if p.FileName() == "note.txt" {
			foundAttachment = true
			// multipart auto-decodes quoted-printable but not base64.
			raw, _ := io.ReadAll(p)
			clean := strings.NewReplacer("\r", "", "\n", "").Replace(string(raw))
			data, err := base64.StdEncoding.DecodeString(clean)
			if err != nil || string(data) != "file body" {
				t.Fatalf("attachment data: %q err=%v", data, err)
			}
		}
	}
	if !foundAttachment {
		t.Fatal("attachment part not found")
	}
}

func TestHeaderInjectionRejected(t *testing.T) {
	// A custom header value with CRLF must not inject a Bcc header.
	msg := Message{
		From:    Address{Email: "a@example.com"},
		To:      []Address{{Email: "b@example.com"}},
		Text:    "x",
		Headers: map[string]string{"X-Trace": "abc\r\nBcc: victim@example.com"},
	}
	raw, err := msg.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m := parseBuilt(t, raw)
	if bcc := m.Header.Get("Bcc"); bcc != "" {
		t.Fatalf("header injection succeeded: Bcc=%q", bcc)
	}
	if x := m.Header.Get("X-Trace"); strings.ContainsAny(x, "\r\n") {
		t.Fatalf("X-Trace still has CRLF: %q", x)
	}
}

func TestAddressControlCharsRejected(t *testing.T) {
	if err := (Address{Email: "a@example.com", Name: "bad\r\nname"}).Validate(); err == nil {
		t.Fatal("expected control-char rejection")
	}
}

func TestHeaderFolding(t *testing.T) {
	many := make([]Address, 0, 20)
	for i := 0; i < 20; i++ {
		many = append(many, Address{Email: "recipient-with-a-longish-name@example.com"})
	}
	msg := Message{From: Address{Email: "a@example.com"}, To: many, Text: "x"}
	raw, err := msg.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 998 {
			t.Fatalf("header line exceeds RFC 5322 hard limit: %d", len(line))
		}
	}
	// It must still parse and expose all recipients.
	m := parseBuilt(t, raw)
	addrs, err := m.Header.AddressList("To")
	if err != nil || len(addrs) != 20 {
		t.Fatalf("To parse: %d addrs err=%v", len(addrs), err)
	}
}

func TestAttachmentNonASCIIFilename(t *testing.T) {
	msg := Message{
		From:        Address{Email: "a@example.com"},
		To:          []Address{{Email: "b@example.com"}},
		Text:        "x",
		Attachments: []Attachment{{Filename: "звіт.txt", Data: []byte("data")}},
	}
	raw, err := msg.build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m := parseBuilt(t, raw)
	_, params, _ := mime.ParseMediaType(m.Header.Get("Content-Type"))
	mr := multipart.NewReader(m.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		if fn := p.FileName(); fn == "звіт.txt" {
			return // decoded RFC 2231 filename correctly
		}
	}
	t.Fatal("non-ASCII filename not round-tripped")
}

func TestBuildRejectsBadAddress(t *testing.T) {
	msg := Message{From: Address{Email: "bad addr"}, To: []Address{{Email: "b@example.com"}}, Text: "x"}
	if _, err := msg.build(); err == nil {
		t.Fatal("expected invalid address error")
	}
}

func TestCaptureSender(t *testing.T) {
	c := NewCapture()
	if err := Send(context.Background(), c, "b@example.com", "hi", "body"); err != nil {
		t.Fatalf("send: %v", err)
	}
	last, ok := c.Last()
	if !ok || last.Subject != "hi" || last.To[0].Email != "b@example.com" {
		t.Fatalf("capture: %+v ok=%v", last, ok)
	}
	if len(c.Messages()) != 1 {
		t.Fatalf("messages: %d", len(c.Messages()))
	}
	c.Reset()
	if len(c.Messages()) != 0 {
		t.Fatal("reset failed")
	}
}

func TestLoggerSender(t *testing.T) {
	var sb strings.Builder
	l := NewLogger(&sb)
	msg := Message{From: Address{Email: "a@example.com"}, To: []Address{{Email: "b@example.com"}}, Subject: "s", Text: "t"}
	if err := l.Send(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(sb.String(), "Subject:") {
		t.Fatalf("logger output missing headers:\n%s", sb.String())
	}
}

func TestSMTPValidation(t *testing.T) {
	s := NewSMTP(SMTPConfig{Host: "localhost"})
	if err := s.Send(context.Background(), Message{To: []Address{{Email: "b@example.com"}}, Text: "x"}); err != ErrNoSender {
		t.Fatalf("expected ErrNoSender, got %v", err)
	}
	s2 := NewSMTP(SMTPConfig{Host: "localhost", From: Address{Email: "a@example.com"}})
	if err := s2.Send(context.Background(), Message{Text: "x"}); err != ErrNoRecipients {
		t.Fatalf("expected ErrNoRecipients, got %v", err)
	}
}

// mockSMTP is a minimal in-process SMTP server for the happy-path send test.
func mockSMTP(t *testing.T, captured *string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		defer ln.Close()
		br := bufio.NewReader(conn)
		writeLine := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }

		writeLine("220 mock ESMTP")
		inData := false
		var data strings.Builder
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					*captured = data.String()
					writeLine("250 OK")
					continue
				}
				data.WriteString(line)
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				writeLine("250 mock")
			case strings.HasPrefix(cmd, "MAIL FROM"), strings.HasPrefix(cmd, "RCPT TO"):
				writeLine("250 OK")
			case strings.HasPrefix(cmd, "DATA"):
				writeLine("354 End data with <CR><LF>.<CR><LF>")
				inData = true
			case strings.HasPrefix(cmd, "QUIT"):
				writeLine("221 Bye")
				return
			default:
				writeLine("250 OK")
			}
		}
	}()
	return ln.Addr().String()
}

func TestSMTPSendHappyPath(t *testing.T) {
	var captured string
	addr := mockSMTP(t, &captured)
	host, portStr, _ := net.SplitHostPort(addr)
	port := 0
	if _, err := fmtSscan(portStr, &port); err != nil {
		t.Fatalf("port: %v", err)
	}

	s := NewSMTP(SMTPConfig{
		Host:    host,
		Port:    port,
		From:    Address{Email: "a@example.com"},
		TLS:     TLSNone,
		Timeout: 3 * time.Second,
	})
	msg := Message{
		To:      []Address{{Email: "b@example.com"}},
		Subject: "hi",
		Text:    "hello body",
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}
	// Give the server goroutine a moment to record DATA.
	deadline := time.Now().Add(time.Second)
	for captured == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !strings.Contains(captured, "hello body") || !strings.Contains(captured, "Subject: hi") {
		t.Fatalf("server did not receive message:\n%s", captured)
	}
}

// fmtSscan avoids importing fmt/strconv just for one parse in the test.
func fmtSscan(s string, p *int) (int, error) {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	*p = n
	return 1, nil
}
