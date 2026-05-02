package telegram

import "context"

type Notifier struct {
	botToken string
	chatID   string
}

func New(botToken, chatID string) *Notifier {
	return &Notifier{botToken: botToken, chatID: chatID}
}

func (n *Notifier) SendIncident(ctx context.Context, text string) error {
	_ = ctx
	_ = text
	_ = n.botToken
	_ = n.chatID
	return nil
}
