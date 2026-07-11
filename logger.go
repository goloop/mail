package mail

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// loggerSender writes each built message to a writer instead of sending it.
// It is handy for local development.
type loggerSender struct {
	mu sync.Mutex
	w  io.Writer
}

// NewLogger returns a Sender that writes the built RFC 5322 message to w.
func NewLogger(w io.Writer) Sender {
	return &loggerSender{w: w}
}

// Send builds the message and writes it to the underlying writer.
func (l *loggerSender) Send(_ context.Context, msg Message) error {
	raw, err := msg.build()
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = fmt.Fprintf(l.w, "--- mail ---\n%s\n", raw)
	return err
}
