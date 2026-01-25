package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Format: gk_RANDOM
	key := generateRandomString(32)
	fullKey := "gk_" + key

	hash, err := bcrypt.GenerateFromPassword([]byte(fullKey), 10)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		return
	}

	fmt.Printf("Key: %s\nHash: %s\n", fullKey, string(hash))
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
