package mail

import (
	"context"
	"sync"
)

// CaptureSender records messages instead of sending them, for tests.
type CaptureSender struct {
	mu       sync.Mutex
	messages []Message
}

// NewCapture returns a CaptureSender.
func NewCapture() *CaptureSender {
	return &CaptureSender{}
}

// Send validates and records a deep copy of the message, so a caller that
// reuses or mutates the original slices (To, Attachments, Attachment.Data)
// after sending cannot change what a test later inspects.
func (c *CaptureSender) Send(_ context.Context, msg Message) error {
	if _, err := msg.build(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, msg.clone())
	return nil
}

// Messages returns a deep copy of the captured messages, so a caller that
// mutates the returned slices, maps or attachment data cannot corrupt the
// captured state a later inspection relies on.
func (c *CaptureSender) Messages() []Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Message, len(c.messages))
	for i, m := range c.messages {
		out[i] = m.clone()
	}
	return out
}

// Last returns a deep copy of the most recently captured message.
func (c *CaptureSender) Last() (Message, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.messages) == 0 {
		return Message{}, false
	}
	return c.messages[len(c.messages)-1].clone(), true
}

// Reset clears the captured messages.
func (c *CaptureSender) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = nil
}
