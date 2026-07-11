package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	"sort"
	"strings"
	"time"
)

// entity is a MIME entity: a leaf (header + body) or a multipart container
// (mediaType + parts).
type entity struct {
	mediaType string // "" for a leaf; e.g. "multipart/mixed" for a container
	header    textproto.MIMEHeader
	body      []byte
	parts     []entity
}

// dateLayout is the RFC 5322 date format.
const dateLayout = "Mon, 02 Jan 2006 15:04:05 -0700"

// build renders the message to RFC 5322/MIME bytes with CRLF line endings. It
// requires at least one recipient, so the dev and test senders reject the same
// empty-recipient message an SMTP server would. The From address may be empty
// here: a transport can fill a default before sending.
func (m Message) build() ([]byte, error) {
	for _, grp := range [][]Address{{m.From}, m.To, m.Cc, m.Bcc, m.ReplyTo} {
		for _, a := range grp {
			if a.Email == "" {
				continue
			}
			if err := a.Validate(); err != nil {
				return nil, fmt.Errorf("mail: invalid address %q: %w", a.Email, err)
			}
		}
	}
	if len(m.recipients()) == 0 {
		return nil, ErrNoRecipients
	}

	root := m.contentEntity()
	if root.header == nil {
		root.header = textproto.MIMEHeader{}
	}
	h := root.header

	if m.From.Email != "" {
		h.Set("From", m.From.String())
	}
	// Set an address header only when it renders to a non-empty list, so a
	// slice of empty addresses never emits a bare "To:" header.
	if s := formatList(m.To); s != "" {
		h.Set("To", s)
	}
	if s := formatList(m.Cc); s != "" {
		h.Set("Cc", s)
	}
	if s := formatList(m.ReplyTo); s != "" {
		h.Set("Reply-To", s)
	}
	if m.Subject != "" {
		// Encode after stripping CR/LF: QEncoding leaves pure ASCII unchanged,
		// so an un-sanitized subject like "hi\r\nBcc: x" would inject a header.
		h.Set("Subject", mime.QEncoding.Encode("utf-8", sanitizeHeaderValue(m.Subject)))
	}
	h.Set("Date", time.Now().Format(dateLayout))
	mid, err := messageID()
	if err != nil {
		return nil, err
	}
	h.Set("Message-Id", mid)
	h.Set("MIME-Version", "1.0")

	// Custom headers must not clobber structural or envelope headers, carry an
	// invalid key, or inject extra headers via CR/LF in the value. Non-ASCII
	// values are RFC 2047 encoded, like the subject and display names.
	for k, v := range m.Headers {
		if isReservedHeader(k) || !validHeaderKey(k) {
			continue
		}
		h.Set(k, mime.QEncoding.Encode("utf-8", sanitizeHeaderValue(v)))
	}

	var buf bytes.Buffer
	if err := render(&buf, root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// contentEntity builds the body, wrapping it in multipart/mixed if there are
// attachments.
func (m Message) contentEntity() entity {
	body := m.bodyEntity()
	if len(m.Attachments) == 0 {
		return body
	}
	parts := make([]entity, 0, len(m.Attachments)+1)
	parts = append(parts, body)
	for _, at := range m.Attachments {
		parts = append(parts, attachmentEntity(at))
	}
	return entity{mediaType: "multipart/mixed", parts: parts}
}

// bodyEntity builds the text/html body: a single part, or multipart/alternative
// when both are present.
func (m Message) bodyEntity() entity {
	hasText, hasHTML := m.Text != "", m.HTML != ""
	switch {
	case hasText && hasHTML:
		return entity{
			mediaType: "multipart/alternative",
			parts:     []entity{textLeaf("text/plain", m.Text), textLeaf("text/html", m.HTML)},
		}
	case hasHTML:
		return textLeaf("text/html", m.HTML)
	default:
		return textLeaf("text/plain", m.Text)
	}
}

// textLeaf builds a quoted-printable text/* leaf.
func textLeaf(ctype, s string) entity {
	return entity{
		header: textproto.MIMEHeader{
			"Content-Type":              {ctype + "; charset=utf-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		},
		body: qpEncode([]byte(s)),
	}
}

// attachmentEntity builds a base64 attachment leaf.
func attachmentEntity(at Attachment) entity {
	ct := at.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	name := sanitizeFilename(at.Filename)

	// mime.FormatMediaType validates the media type and RFC 2231 encodes a
	// non-ASCII filename, returning "" on an invalid type.
	ctHeader := mime.FormatMediaType(ct, map[string]string{"name": name})
	if ctHeader == "" {
		ctHeader = mime.FormatMediaType("application/octet-stream", map[string]string{"name": name})
	}
	cdHeader := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if cdHeader == "" {
		cdHeader = "attachment"
	}

	return entity{
		header: textproto.MIMEHeader{
			"Content-Type":              {ctHeader},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {cdHeader},
		},
		body: base64Wrap(at.Data),
	}
}

// render writes the entity (leaf or multipart) with CRLF line endings. It
// returns an error only if a random multipart boundary cannot be generated.
func render(buf *bytes.Buffer, e entity) error {
	if e.mediaType != "" {
		boundary, err := randomBoundary()
		if err != nil {
			return err
		}
		if e.header == nil {
			e.header = textproto.MIMEHeader{}
		}
		e.header.Set("Content-Type", fmt.Sprintf("%s; boundary=%q", e.mediaType, boundary))
		writeHeaders(buf, e.header)
		buf.WriteString("\r\n")
		for _, p := range e.parts {
			buf.WriteString("--" + boundary + "\r\n")
			if err := render(buf, p); err != nil {
				return err
			}
			buf.WriteString("\r\n")
		}
		buf.WriteString("--" + boundary + "--\r\n")
		return nil
	}
	writeHeaders(buf, e.header)
	buf.WriteString("\r\n")
	buf.Write(e.body)
	return nil
}

// headerOrder is the canonical (textproto-canonicalized) header order.
var headerOrder = []string{
	"From", "To", "Cc", "Reply-To", "Subject", "Date", "Message-Id",
	"Mime-Version", "Content-Type", "Content-Transfer-Encoding",
	"Content-Disposition",
}

// writeHeaders writes headers in a stable order for deterministic output.
func writeHeaders(buf *bytes.Buffer, h textproto.MIMEHeader) {
	seen := make(map[string]bool, len(h))
	for _, k := range headerOrder {
		if vs, ok := h[k]; ok {
			for _, v := range vs {
				writeHeaderLine(buf, k, v)
			}
			seen[k] = true
		}
	}
	rest := make([]string, 0, len(h))
	for k := range h {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		for _, v := range h[k] {
			writeHeaderLine(buf, k, v)
		}
	}
}

// maxHeaderLine is the soft line length used when folding header values; it
// stays well under the RFC 5322 hard limit of 998 octets.
const maxHeaderLine = 76

// writeHeaderLine writes "Key: value", folding long values at whitespace with
// CRLF + space so no line exceeds the soft limit.
func writeHeaderLine(buf *bytes.Buffer, key, value string) {
	if len(key)+2+len(value) <= maxHeaderLine {
		fmt.Fprintf(buf, "%s: %s\r\n", key, value)
		return
	}
	buf.WriteString(key)
	buf.WriteByte(':')
	lineLen := len(key) + 1
	for _, word := range strings.Fields(value) {
		if lineLen > len(key)+1 && lineLen+1+len(word) > maxHeaderLine {
			buf.WriteString("\r\n ")
			lineLen = 1
		} else {
			buf.WriteByte(' ')
			lineLen++
		}
		buf.WriteString(word)
		lineLen += len(word)
	}
	buf.WriteString("\r\n")
}

// validHeaderKey reports whether k is a valid RFC 5322 field name (printable
// ASCII, no colon, no whitespace).
func validHeaderKey(k string) bool {
	if k == "" {
		return false
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		if c < '!' || c > '~' || c == ':' {
			return false
		}
	}
	return true
}

// sanitizeHeaderValue strips CR and LF to prevent header injection.
func sanitizeHeaderValue(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

// isReservedHeader reports whether a custom header would collide with a
// structural or envelope header.
func isReservedHeader(k string) bool {
	switch textproto.CanonicalMIMEHeaderKey(k) {
	case "From", "To", "Cc", "Bcc", "Reply-To", "Subject", "Date",
		"Message-Id", "Mime-Version", "Content-Type",
		"Content-Transfer-Encoding", "Content-Disposition":
		return true
	default:
		return false
	}
}

// qpEncode quoted-printable encodes b (CRLF line endings).
func qpEncode(b []byte) []byte {
	var buf bytes.Buffer
	w := quotedprintable.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

// base64Wrap base64-encodes data, wrapped at 76 columns per RFC 2045.
func base64Wrap(data []byte) []byte {
	enc := base64.StdEncoding.EncodeToString(data)
	var buf bytes.Buffer
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		buf.WriteString(enc[i:end])
		buf.WriteString("\r\n")
	}
	return buf.Bytes()
}

// maxFilenameBytes bounds an attachment filename so it cannot produce a
// Content-Type/Content-Disposition line too long for a MIME parser to accept.
// Header folding does not break a single long unspaced token, so the name
// itself must be bounded.
const maxFilenameBytes = 200

// sanitizeFilename removes characters that could break a header line and bounds
// the length, preserving a short extension.
func sanitizeFilename(name string) string {
	name = strings.NewReplacer("\r", "", "\n", "", `"`, "", "\\", "", "/", "_").Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "attachment"
	}
	return truncateFilename(name, maxFilenameBytes)
}

// truncateFilename shortens name to at most max bytes, keeping a short trailing
// extension and never cutting in the middle of a UTF-8 rune.
func truncateFilename(name string, max int) string {
	if len(name) <= max {
		return name
	}
	ext := ""
	if i := strings.LastIndexByte(name, '.'); i > 0 && len(name)-i <= 16 {
		ext = name[i:]
	}
	keep := max - len(ext)
	if keep < 1 {
		keep, ext = max, ""
	}
	base := strings.ToValidUTF8(name[:keep], "")
	return base + ext
}

// randomBoundary returns a random MIME multipart boundary. It errors if the
// system random source fails, so a boundary is never silently deterministic.
func randomBoundary() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("mail: random boundary: %w", err)
	}
	return "goloop-" + hex.EncodeToString(b[:]), nil
}

// messageID returns a unique RFC 5322 Message-ID. It errors if the system
// random source fails, so the ID is never silently deterministic.
func messageID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("mail: message id: %w", err)
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(b[:]), host), nil
}
