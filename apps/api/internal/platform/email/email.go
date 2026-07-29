package email

import "context"

type TransactionalEmail struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type TransactionalEmailSender interface {
	Send(ctx context.Context, message TransactionalEmail) error
}
