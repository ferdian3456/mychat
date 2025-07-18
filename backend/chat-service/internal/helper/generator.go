package helper

import (
	"crypto/sha256"
	"encoding/hex"
)

func GenerateSHA256Hash(value string) string {
	data := sha256.New()
	data.Write([]byte(value))
	hashedValue := data.Sum(nil)

	hashedValueHex := hex.EncodeToString(hashedValue)

	return hashedValueHex
}
