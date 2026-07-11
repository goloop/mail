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

// clone returns a deep copy: the address slices, the headers map and each
// attachment's bytes are copied, so mutating the original after cloning cannot
// change the copy.
func (m Message) clone() Message {
	cp := m
	cp.To = append([]Address(nil), m.To...)
	cp.Cc = append([]Address(nil), m.Cc...)
	cp.Bcc = append([]Address(nil), m.Bcc...)
	cp.ReplyTo = append([]Address(nil), m.ReplyTo...)
	if m.Attachments != nil {
		cp.Attachments = make([]Attachment, len(m.Attachments))
		for i, a := range m.Attachments {
			a.Data = append([]byte(nil), a.Data...)
			cp.Attachments[i] = a
		}
	}
	if m.Headers != nil {
		cp.Headers = make(map[string]string, len(m.Headers))
		for k, v := range m.Headers {
			cp.Headers[k] = v
		}
	}
	return cp
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
