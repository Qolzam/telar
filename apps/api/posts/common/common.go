package common

import (
    "crypto/rand"
    "fmt"
    "strings"
    "time"
)

const contentMaxLength = 20

const charset = "abcdefghijklmnopqrstuvwxyz" +
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// stringWithCharset generates a cryptographically secure random string
func stringWithCharset(length int, charset string) string {
    b := make([]byte, length)
    charsetBytes := []byte(charset)
    
    // Generate random bytes
    randomBytes := make([]byte, length)
    if _, err := rand.Read(randomBytes); err != nil {
        // Fallback to timestamp-based generation if crypto/rand fails
        fallbackSeed := time.Now().UnixNano()
        for i := range b {
            fallbackSeed = (fallbackSeed * 1103515245 + 12345) & 0x7fffffff
            b[i] = charsetBytes[fallbackSeed%int64(len(charsetBytes))]
        }
        return string(b)
    }
    
    // Use crypto random bytes to select from charset
    for i := range b {
        b[i] = charsetBytes[int(randomBytes[i])%len(charsetBytes)]
    }
    return string(b)
}

// StringRand generates a random string of given length
func StringRand(length int) string {
    return stringWithCharset(length, charset)
}

// GeneratePostURLKey replicates original url-key generation logic
func GeneratePostURLKey(socialName, body, postId string) string {
    content := body
    if contentMaxLength <= len(body) {
        content = body[:contentMaxLength]
    }
    return strings.ToLower(fmt.Sprintf("%s_%s-post-%s-%s", socialName, strings.ReplaceAll(content, " ", "-"), strings.Split(postId, "-")[0], StringRand(5)))
}

// SplitAndTrim splits a string by sep and trims whitespace; empty parts removed
func SplitAndTrim(s string, sep string) []string {
    parts := strings.Split(s, sep)
    out := make([]string, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p != "" {
            out = append(out, p)
        }
    }
    return out
}

