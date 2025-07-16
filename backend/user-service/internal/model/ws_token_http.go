package model

type WebsocketTokenResponse struct {
	WebsocketToken          string `json:"websocket_token"`
	TokenType               string `json:"token_type"`
	WebsocketTokenExpiresIn int    `json:"websocket_token_expires_in"`
}
