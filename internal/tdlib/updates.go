package tdlib

import (
	"encoding/json"
	"fmt"
	"time"
)

func WaitForUpdateType(tdjson *TDJSON, clientID int32, wantedType string, timeout time.Duration) (string, error) {
	d := dispatcherFor(tdjson)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case update := <-d.updates:
			var envelope struct {
				Type string `json:"@type"`
			}
			if err := json.Unmarshal([]byte(update), &envelope); err != nil {
				continue
			}
			if envelope.Type == wantedType {
				return update, nil
			}
		case <-time.After(time.Until(deadline)):
		case <-d.stopCh:
			return "", fmt.Errorf("TDLib client closed")
		}
	}

	return "", fmt.Errorf("timed out waiting for update type %s", wantedType)
}

type NewMessageUpdate struct {
	Type    string `json:"@type"`
	Message struct {
		ID      int64 `json:"id"`
		ChatID  int64 `json:"chat_id"`
		Content struct {
			Type string `json:"@type"`
			Text struct {
				Text string `json:"text"`
			} `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

func ReceiveUpdates(tdjson *TDJSON) (string, error) {
	d := dispatcherFor(tdjson)
	select {
	case update := <-d.updates:
		return update, nil
	case <-time.After(5 * time.Second):
		return "", nil
	case <-d.stopCh:
		return "", nil
	}
}

func ParseNewMessage(updateJSON string) (*NewMessageUpdate, bool) {
	var u NewMessageUpdate
	if err := json.Unmarshal([]byte(updateJSON), &u); err != nil {
		return nil, false
	}
	if u.Type != "updateNewMessage" {
		return nil, false
	}
	if u.Message.Content.Type != "messageText" {
		return nil, false
	}
	return &u, true
}

type UpdateNewMessage struct {
	Type    string  `json:"@type"`
	Message Message `json:"message"`
}

type Message struct {
	ID       int64    `json:"id"`
	ChatID   int64    `json:"chat_id"`
	Date     int64    `json:"date"`
	SenderID SenderID `json:"sender_id"`
	Content  Content  `json:"content"`
}

type SenderID struct {
	Type   string `json:"@type"`
	UserID int64  `json:"user_id,omitempty"`
	ChatID int64  `json:"chat_id,omitempty"`
}

type Content struct {
	Type string `json:"@type"`
	Text struct {
		Text string `json:"text"`
	} `json:"text,omitempty"`
}

func ParseUpdateNewMessage(updateJSON string) (*UpdateNewMessage, bool) {
	var u UpdateNewMessage
	if err := json.Unmarshal([]byte(updateJSON), &u); err != nil {
		return nil, false
	}
	if u.Type != "updateNewMessage" {
		return nil, false
	}
	if u.Message.Content.Type != "messageText" {
		return nil, false
	}
	return &u, true
}
