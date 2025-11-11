package common

import "testing"

func TestGeneratePostURLKey_Basic(t *testing.T) {
    key := GeneratePostURLKey("john", "hello world example body", "123e4567-e89b-12d3-a456-426614174000")
    if key == "" { t.Fatalf("empty key") }
    if len(key) < 10 { t.Fatalf("key too short: %s", key) }
}


