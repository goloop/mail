package mail

import (
	"fmt"
	"net/mail"
	"strings"
)

// Address is an email address with an optional display name.
type Address struct {
	Name  string
	Email string
}

// String formats the address per RFC 5322, encoding a non-ASCII display name
// as an RFC 2047 encoded-word: `Name <email>` or just `email`.
func (a Address) String() string {
	m := mail.Address{Name: a.Name, Address: a.Email}
	return m.String()
}

// Validate reports whether the address has a non-empty, parseable email. It
// rejects control characters outright so validation cannot be bypassed by
// characters that formatting would silently drop.
func (a Address) Validate() error {
	if strings.TrimSpace(a.Email) == "" {
		return ErrEmptyAddress
	}
	if hasControl(a.Email) || hasControl(a.Name) {
		return fmt.Errorf("mail: address %q contains control characters", a.Email)
	}
	if _, err := mail.ParseAddress(a.String()); err != nil {
		return err
	}
	return nil
}

// hasControl reports whether s contains an ASCII control character.
func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// formatList joins addresses with ", " in RFC 5322 form.
func formatList(as []Address) string {
	parts := make([]string, 0, len(as))
	for _, a := range as {
		if a.Email != "" {
			parts = append(parts, a.String())
		}
	}
	return strings.Join(parts, ", ")
}
