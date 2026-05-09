package gameops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

type Principal struct {
	UserID string
	Role   string
}

var ErrUnauthorized = errors.New("unauthorized")

func (s *Server) issueToken(userID, role string) string {
	expires := nowMS() + int64((2 * time.Hour).Milliseconds())
	payload := userID + "|" + role + "|" + formatInt(expires)
	mac := hmac.New(sha256.New, []byte(s.cfg.TokenSigningKey))
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + signature
}

func (s *Server) verifyToken(token string) (*Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrUnauthorized
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrUnauthorized
	}
	payload := string(payloadBytes)
	mac := hmac.New(sha256.New, []byte(s.cfg.TokenSigningKey))
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, ErrUnauthorized
	}
	fields := strings.Split(payload, "|")
	if len(fields) != 3 {
		return nil, ErrUnauthorized
	}
	expires, err := parseInt(fields[2])
	if err != nil || expires < nowMS() {
		return nil, ErrUnauthorized
	}
	return &Principal{UserID: fields[0], Role: fields[1]}, nil
}

func formatInt(value int64) string {
	return strconvFormatInt(value)
}

func parseInt(value string) (int64, error) {
	return strconvParseInt(value)
}
