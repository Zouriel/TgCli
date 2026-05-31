package tdlib

import (
	"encoding/json"
	"fmt"
	"time"
)

func WaitForUpdateType(tdjson *TDJSON, clientID int32, wantedType string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		update, err := tdjson.Receive(1.0)
		if err != nil {
			continue
		}
		if update == "" {
			continue
		}

		var envelope struct {
			Type string `json:"@type"`
		}
		if err := json.Unmarshal([]byte(update), &envelope); err != nil {
			continue
		}

		if envelope.Type == wantedType {
			return update, nil
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
	receiveMu.Lock()
	update, err := tdjson.Receive(5.0)
	receiveMu.Unlock()
	return update, err
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
