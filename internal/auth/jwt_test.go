package auth

import (
	"testing"
	"time"
)

func TestJWT_MakeAndParse(t *testing.T) {

	t.Setenv("JWT_SECRET", "testsecret") // фиксированный ключ

	tok, err := MakeToken(7, "alice", 2*time.Second)
	if err != nil {
		t.Fatalf("MakeToken error: %v", err)
	}
	claims, err := ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if got := int64(claims["sub"].(float64)); got != 7 {
		t.Fatalf("sub mismatch: got %d", got)
	}
	if got := claims["login"].(string); got != "alice" {
		t.Fatalf("login mismatch: %s", got)
	}
}

func TestJWT_Expired(t *testing.T) {

	t.Setenv("JWT_SECRET", "testsecret")

	tok, err := MakeToken(1, "u", -1*time.Second) // уже просрочен
	if err != nil {
		t.Fatalf("MakeToken error: %v", err)
	}
	if _, err := ParseToken(tok); err == nil {
		t.Fatalf("expected error for expired token")
	}
}
