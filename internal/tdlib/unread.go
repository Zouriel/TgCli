package tdlib

import (
	"encoding/json"
	"fmt"
	"time"
)

// ChatInfo carries the unread state we need to triage a chat.
type ChatInfo struct {
	ID                     int64
	Title                  string
	TypeName               string // "private" | "group" | "other"
	UserID                 int64  // for private chats: the other user
	UnreadCount            int
	LastReadInboxMessageID int64
}

// FetchChatInfo returns a chat's unread state (from the local cache populated by
// getChats — fast, no network).
func FetchChatInfo(tdjson *TDJSON, clientID int32, chatID int64) (ChatInfo, error) {
	req := fmt.Sprintf(`{"@type":"getChat","chat_id":%d}`, chatID)
	resp, err := SendRequestAndWait(tdjson, clientID, req, "get-chat-info", 10*time.Second)
	if err != nil {
		return ChatInfo{}, err
	}

	var out struct {
		ID                     int64  `json:"id"`
		Title                  string `json:"title"`
		UnreadCount            int    `json:"unread_count"`
		LastReadInboxMessageID int64  `json:"last_read_inbox_message_id"`
		Type                   struct {
			Type   string `json:"@type"`
			UserID int64  `json:"user_id"`
		} `json:"type"`
	}
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return ChatInfo{}, err
	}

	info := ChatInfo{
		ID:                     out.ID,
		Title:                  out.Title,
		UserID:                 out.Type.UserID,
		UnreadCount:            out.UnreadCount,
		LastReadInboxMessageID: out.LastReadInboxMessageID,
	}
	switch out.Type.Type {
	case "chatTypePrivate", "chatTypeSecret":
		info.TypeName = "private"
	case "chatTypeBasicGroup", "chatTypeSupergroup":
		info.TypeName = "group"
	default:
		info.TypeName = "other"
	}
	return info, nil
}

// FetchRecentMessages returns up to limit most-recent messages for a chat,
// newest first, without opening the chat or marking anything read.
func FetchRecentMessages(tdjson *TDJSON, clientID int32, chatID int64, limit int) ([]Message, error) {
	req := fmt.Sprintf(`{"@type":"getChatHistory","chat_id":%d,"from_message_id":0,"offset":0,"limit":%d,"only_local":false}`, chatID, limit)
	resp, err := SendRequestAndWait(tdjson, clientID, req, "get-recent", 10*time.Second)
	if err != nil {
		return nil, err
	}
	var out ChatHistory
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}

// MarkMessagesRead marks the given messages read (and sends read receipts), so
// they won't be triaged again.
func MarkMessagesRead(tdjson *TDJSON, clientID int32, chatID int64, messageIDs []int64) error {
	if len(messageIDs) == 0 {
		return nil
	}
	req := map[string]any{
		"@type":       "viewMessages",
		"chat_id":     chatID,
		"message_ids": messageIDs,
		"force_read":  true,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = SendRequestAndWait(tdjson, clientID, string(b), "view-messages", 10*time.Second)
	return err
}
