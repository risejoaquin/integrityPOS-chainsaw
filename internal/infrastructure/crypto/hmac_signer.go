package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func Sign(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func Verify(secret, message, signature string) (bool, error) {
	expected := Sign(secret, message)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return false, errors.New("invalid signature")
	}
	return true, nil
}
