// Package auth provides password hashing, JWT issuance/verification and
// opaque refresh-token helpers.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("invalid token")

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewManager(secret []byte, accessTTL, refreshTTL time.Duration, issuer string) *Manager {
	return &Manager{secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL, issuer: issuer}
}

func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }
func (m *Manager) AccessTTL() time.Duration  { return m.accessTTL }

// ---- Passwords (bcrypt, cost 12) ----

func HashPassword(pw string) (string, error) {
	if len(pw) < 8 {
		return "", errors.New("password too short")
	}
	// bcrypt silently truncates at 72 bytes; reject longer to avoid surprises.
	if len(pw) > 72 {
		return "", errors.New("password too long (max 72 bytes)")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// ---- Access tokens (JWT) ----

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (m *Manager) IssueAccessToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(m.secret)
}

func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ---- Refresh tokens (opaque, only hash stored) ----

// NewRefreshToken returns a high-entropy opaque token and its sha256 hash.
func NewRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	hash = HashToken(token)
	return token, hash, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomString returns a URL-safe random string with n bytes of entropy.
func RandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
