package mail

// Attachment is a file attached to a message.
type Attachment struct {
	Filename    string
	ContentType string // defaults to application/octet-stream
	Data        []byte
}

// Message is an email to be built and sent. Text and/or HTML set the body;
// providing both yields a multipart/alternative message. Any attachments wrap
// the body in multipart/mixed.
type Message struct {
	From        Address
	To          []Address
	Cc          []Address
	Bcc         []Address
	ReplyTo     []Address
	Subject     string
	Text        string
	HTML        string
	Attachments []Attachment
	Headers     map[string]string // extra headers (structural headers win)
}

// recipients returns the envelope recipient emails (To + Cc + Bcc).
func (m Message) recipients() []string {
	out := make([]string, 0, len(m.To)+len(m.Cc)+len(m.Bcc))
	for _, grp := range [][]Address{m.To, m.Cc, m.Bcc} {
		for _, a := range grp {
			if a.Email != "" {
				out = append(out, a.Email)
			}
		}
	}
	return out
}
