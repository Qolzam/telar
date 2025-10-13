package login

import (
	"testing"

	"github.com/qolzam/telar/apps/api/internal/utils"
)

func TestComparePassword_Mismatch(t *testing.T) {
	s := &Service{config: &ServiceConfig{}}
	hash, _ := utils.Hash("secret")
	if s.ComparePassword(hash, "wrong") == nil {
		t.Fatalf("expected mismatch error")
	}
}

func TestComparePassword_Match(t *testing.T) {
	s := &Service{config: &ServiceConfig{}}
	hash, _ := utils.Hash("secret")
	if s.ComparePassword(hash, "secret") != nil {
		t.Fatalf("expected match")
	}
}
