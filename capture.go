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

// Send validates and records the message.
func (c *CaptureSender) Send(_ context.Context, msg Message) error {
	if _, err := msg.build(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, msg)
	return nil
}

// Messages returns a copy of the captured messages.
func (c *CaptureSender) Messages() []Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Message(nil), c.messages...)
}

// Last returns the most recently captured message.
func (c *CaptureSender) Last() (Message, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.messages) == 0 {
		return Message{}, false
	}
	return c.messages[len(c.messages)-1], true
}

// Reset clears the captured messages.
func (c *CaptureSender) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = nil
}
