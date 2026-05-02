package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Notifier struct {
	botToken string
	chatID   string
}

func New(botToken, chatID string) *Notifier {
	return &Notifier{botToken: botToken, chatID: chatID}
}

func (n *Notifier) SendIncident(ctx context.Context, text string) error {
	if n.botToken == "" || n.chatID == "" {
		return nil
	}
	body, err := json.Marshal(map[string]string{
		"chat_id": n.chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+n.botToken+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage status %s", resp.Status)
	}
	return nil
}
