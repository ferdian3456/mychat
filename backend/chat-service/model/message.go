package model

import "time"

type Message struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Text           string    `json:"text"`
	CreatedAt      time.Time `json:"created_at"`
}

type IncomingMessage struct {
	ConversationID int    `json:"conversation_id"`
	Text           string `json:"text"`
}
