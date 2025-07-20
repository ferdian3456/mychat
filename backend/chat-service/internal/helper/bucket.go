package helper

import (
	"encoding/json"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/model"
	"hash/crc32"
)

func GetBucketForUser(userID string, bucketCount int) int {
	hash := crc32.ChecksumIEEE([]byte(userID))
	return int(hash % uint32(bucketCount))
}

func MessageBelongsToUser(payload string, userID string) bool {
	// Simplified: decode and check if user is a participant
	var msg model.Message
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return false
	}
	return msg.SenderID == userID
}
