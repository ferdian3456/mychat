package model

type UserConversationRequest struct {
	ParticipantIDs []string `json:"participant_ids"`
}

type UserAddConversationRequest struct {
	Username string `json:"username"`
}

type UserAllConversationIDResponse struct {
	ConversationID int    `json:"conversation_id"`
	Username       string `json:"username"`
}

type UserConversationResponse struct {
	ConversationID int `json:"conversation_id"`
}
