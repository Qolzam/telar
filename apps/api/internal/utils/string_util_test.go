package utils

import "testing"

func TestLowerFirstAndTypeHelpers(t *testing.T) {
    if LowerFirst("Hello") != "hello" { t.Fatalf("lowerFirst failed") }
    if GetType("*pkg.Type") != "Type" { t.Fatalf("getType failed") }
}

func TestGenerateDigits_Length(t *testing.T) {
    d := GenerateDigits(6)
    if len(d) != 6 { t.Fatalf("expected 6 digits, got %d", len(d)) }
}


