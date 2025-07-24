package model

import "time"

type Message struct {
	ID             string    `json:"id"`
	ConversationID int       `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	RecipientIDs   []string  `json:"recipient_ids"`
	Text           string    `json:"text"`
	CreatedAt      time.Time `json:"created_at"`
}

type MessagePagination struct {
	Message    []Message `json:"message"`
	NextCursor *string   `json:"next_cursor"`
	HasMore    bool      `json:"has_more"`
}

type IncomingMessage struct {
	ConversationID int    `json:"conversation_id"`
	Text           string `json:"text"`
}
