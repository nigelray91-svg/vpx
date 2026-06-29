package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "correct horse battery") {
		t.Fatal("expected password to verify")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password to fail")
	}
	if CheckPassword("", "anything") {
		t.Fatal("empty hash must never verify")
	}
}

func TestPasswordLengthBounds(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := HashPassword(string(long)); err == nil {
		t.Fatal("expected >72 byte password to be rejected")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	m := NewManager([]byte("0123456789abcdef0123456789abcdef"), 15*time.Minute, time.Hour, "vpx")
	id := uuid.New()
	tok, err := m.IssueAccessToken(id, "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := m.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != id.String() {
		t.Fatalf("subject mismatch: %s", claims.Subject)
	}
	if claims.Role != "admin" {
		t.Fatalf("role mismatch: %s", claims.Role)
	}
}

func TestJWTRejectsWrongSecret(t *testing.T) {
	m1 := NewManager([]byte("0123456789abcdef0123456789abcdef"), time.Minute, time.Hour, "vpx")
	m2 := NewManager([]byte("ffffffffffffffffffffffffffffffff"), time.Minute, time.Hour, "vpx")
	tok, _ := m1.IssueAccessToken(uuid.New(), "user")
	if _, err := m2.ParseAccessToken(tok); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
}

func TestRefreshTokenHashStable(t *testing.T) {
	tok, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("new refresh: %v", err)
	}
	if HashToken(tok) != hash {
		t.Fatal("hash of token must match returned hash")
	}
	if tok == hash {
		t.Fatal("token and hash must differ")
	}
}
