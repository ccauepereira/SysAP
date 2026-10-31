package email

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type LocalSMTPConfig struct {
	Host        string
	Port        string
	SenderEmail string
	SenderName  string
	Timeout     time.Duration
}

type LocalSMTPEmailProvider struct {
	address     string
	host        string
	senderEmail string
	senderName  string
	timeout     time.Duration
}

func NewLocalSMTPEmailProvider(cfg LocalSMTPConfig) (*LocalSMTPEmailProvider, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return nil, fmt.Errorf("local SMTP host must be loopback")
	}
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "54325"
	}
	senderEmail := strings.TrimSpace(cfg.SenderEmail)
	if senderEmail == "" {
		senderEmail = "nao-responder@sysap.local"
	}
	if _, err := mail.ParseAddress(senderEmail); err != nil {
		return nil, fmt.Errorf("invalid local SMTP sender email")
	}
	senderName := strings.TrimSpace(cfg.SenderName)
	if senderName == "" {
		senderName = "SysAP Artur Performance"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &LocalSMTPEmailProvider{
		address:     net.JoinHostPort(host, port),
		host:        host,
		senderEmail: senderEmail,
		senderName:  senderName,
		timeout:     timeout,
	}, nil
}

func (p *LocalSMTPEmailProvider) Send(ctx context.Context, msg TransactionalEmail) error {
	recipient := strings.TrimSpace(msg.To)
	if _, err := mail.ParseAddress(recipient); err != nil {
		return sendFailure("email: destinatario invalido", "failed")
	}
	if strings.TrimSpace(msg.Subject) == "" || strings.TrimSpace(msg.Text) == "" {
		return sendFailure("email: conteudo obrigatorio ausente", "failed")
	}

	dialCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", p.address)
	if err != nil {
		return sendFailure("email: falha na comunicacao com servidor SMTP local", "unknown")
	}
	defer conn.Close()

	if deadline, ok := dialCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return sendFailure("email: falha na comunicacao com servidor SMTP local", "unknown")
	}
	defer client.Close()

	if err := client.Mail(p.senderEmail); err != nil {
		return sendFailure("email: servidor SMTP rejeitou o remetente", "failed")
	}
	if err := client.Rcpt(recipient); err != nil {
		return sendFailure("email: servidor SMTP rejeitou o destinatario", "failed")
	}

	wc, err := client.Data()
	if err != nil {
		return sendFailure("email: falha ao transmitir mensagem SMTP", "unknown")
	}

	cleanSubject := strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(msg.Subject))

	var body strings.Builder
	body.WriteString(fmt.Sprintf("From: %s <%s>\r\n", p.senderName, p.senderEmail))
	body.WriteString(fmt.Sprintf("To: <%s>\r\n", recipient))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", cleanSubject))
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(msg.Text)
	body.WriteString("\r\n")

	if _, err := wc.Write([]byte(body.String())); err != nil {
		_ = wc.Close()
		return sendFailure("email: falha ao gravar corpo SMTP", "unknown")
	}
	if err := wc.Close(); err != nil {
		return sendFailure("email: falha ao concluir envio SMTP", "unknown")
	}
	_ = client.Quit()
	return nil
}
