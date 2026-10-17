package mail

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestCancelSMTPGreeting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := listener.Accept()
		if err == nil {
			accepted <- c
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := smtpSender{cfg: SMTPConfig{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, TLS: TLSNone, Timeout: 10 * time.Second}}
	done := make(chan error, 1)
	go func() {
		c, err := sender.dial(ctx)
		if c != nil {
			c.Close()
		}
		done <- err
	}()
	select {
	case c := <-accepted:
		defer c.Close()
	case <-time.After(3 * time.Second):
		t.Fatal("connection not accepted")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled greeting succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation did not interrupt greeting")
	}
}
