package sharedUtils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

func ComputeAPIKeyHash(rawKey string) (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)
	input := salt + ":" + rawKey
	hash := sha256.Sum256([]byte(input))
	return salt + ":" + hex.EncodeToString(hash[:]), nil
}

func VerifyAPIKeyHash(rawKey string, stored string) bool {
	parts := strings.Split(stored, ":")
	if len(parts) != 2 {
		return false
	}
	salt := parts[0]
	expectedHash := parts[1]
	input := salt + ":" + rawKey
	hash := sha256.Sum256([]byte(input))
	computed := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHash)) == 1
}

func ValidatePermissions[T any](userPermissions []T, requested []string, extractUID func(T) string) error {
	allowed := make(map[string]bool, len(userPermissions))
	for _, p := range userPermissions {
		allowed[extractUID(p)] = true
	}
	for _, p := range requested {
		if !allowed[p] {
			return fmt.Errorf("permission '%s' is not allowed", p)
		}
	}
	return nil
}
