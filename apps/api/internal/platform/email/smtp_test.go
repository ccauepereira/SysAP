package email

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewLocalSMTPEmailProvider(t *testing.T) {
	if _, err := NewLocalSMTPEmailProvider(LocalSMTPConfig{Host: "example.com"}); err == nil {
		t.Fatal("expected non-loopback host to be rejected")
	}
	if _, err := NewLocalSMTPEmailProvider(LocalSMTPConfig{SenderEmail: "invalid"}); err == nil {
		t.Fatal("expected invalid sender email to be rejected")
	}
	p, err := NewLocalSMTPEmailProvider(LocalSMTPConfig{})
	if err != nil || p == nil {
		t.Fatalf("expected valid default local SMTP provider, got err=%v", err)
	}
}

func TestLocalSMTPEmailProvider_Send(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	received := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		_, _ = conn.Write([]byte("220 localhost ESMTP\r\n"))
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" || line == ".\n" {
					inData = false
					_, _ = conn.Write([]byte("250 OK\r\n"))
					continue
				}
				data.WriteString(line)
				continue
			}
			upper := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				_, _ = conn.Write([]byte("250-localhost\r\n250 OK\r\n"))
			case strings.HasPrefix(upper, "MAIL FROM:"):
				_, _ = conn.Write([]byte("250 OK\r\n"))
			case strings.HasPrefix(upper, "RCPT TO:"):
				_, _ = conn.Write([]byte("250 OK\r\n"))
			case strings.HasPrefix(upper, "DATA"):
				inData = true
				_, _ = conn.Write([]byte("354 End data with <CR><LF>.<CR><LF>\r\n"))
			case strings.HasPrefix(upper, "QUIT"):
				_, _ = conn.Write([]byte("221 Bye\r\n"))
				received <- data.String()
				return
			}
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	p, err := NewLocalSMTPEmailProvider(LocalSMTPConfig{
		Host:    "127.0.0.1",
		Port:    port,
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	if err := p.Send(context.Background(), TransactionalEmail{
		To:      "atleta@example.test",
		Subject: "Matricula SysAP",
		Text:    "Sua matricula: 2026000001",
	}); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	select {
	case msg := <-received:
		if !strings.Contains(msg, "2026000001") {
			t.Fatalf("expected body in SMTP payload, got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP message")
	}

	err = p.Send(context.Background(), TransactionalEmail{
		To:      "invalid-recipient",
		Subject: "Test",
		Text:    "Body",
	})
	var classified interface{ DeliveryOutcome() string }
	if !errors.As(err, &classified) || classified.DeliveryOutcome() != "failed" {
		t.Fatalf("expected permanent failed classification for invalid recipient, got %v", err)
	}
}
