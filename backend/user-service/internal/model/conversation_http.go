package model

type UserConversationRequest struct {
	ParticipantIDs []string `json:"participant_ids"`
}

type UserConversationResponse struct {
	ConversationID int `json:"conversation_id"`
}
