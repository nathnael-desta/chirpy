package auth

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHashPasswordAndCheckPasswordHash(t *testing.T) {
	password := "testpassword123"

	// Hash the password
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword returned an empty string")
	}

	// CheckPasswordHash should succeed with correct password
	if err := CheckPasswordHash(password, hash); err != nil {
		t.Fatalf("CheckPasswordHash returned error with correct password: %v", err)
	}

	// CheckPasswordHash should fail with wrong password
	wrongPassword := "nottherightpassword"
	if err := CheckPasswordHash(wrongPassword, hash); err == nil {
		t.Fatal("CheckPasswordHash should return error with incorrect password, but returned nil")
	}
}

func TestHashPasswordErrorHandling(t *testing.T) {
	// bcrypt accepts any string, but let's try an empty password for completeness
	_, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword returned error for empty password: %v", err)
	}
}

func TestMakeJWT(t *testing.T) {
	// check if you can create a jwt
	// userID uuid.UUID, tokenSecret string, expiresIn time.Duration
	userId, err := uuid.Parse("2945fef6-2470-4cc0-ab15-38608dfc74b8")
	tokenSecret := "f1d9cffa3564f3d1e75027ec2805382a"
	if err != nil {
		t.Fatalf("failed to make uuid: %s", err)
	}
	tokenString, err := MakeJWT(userId, tokenSecret, time.Second)
	if err != nil {
		t.Fatalf("makeJWT returned an error %s: ", err)
	}

	if tokenString == "" {
		t.Fatalf("makeJWT returned an empty string as the tokenstring")
	}
}

func TestValidateJWT(t *testing.T) {
	// check if you can create a jwt
	// userID uuid.UUID, tokenSecret string, expiresIn time.Duration
	// ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	tokenSecret := "f1d9cffa3564f3d1e75027ec2805382a"
	userIdAnswer, err := uuid.Parse("2945fef6-2470-4cc0-ab15-38608dfc74b8")
	if err != nil {
		t.Fatalf("failed to make uuid: %s", err)
	}
	tokenString, err := MakeJWT(userIdAnswer, tokenSecret, time.Hour*24)
	if err != nil {
		t.Fatalf("failed to make jwt: %s", err)
	}

	userID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Fatalf("ValidateJWT returned an error: %s ", err)
	}

	if userID != userIdAnswer {
		t.Fatalf("makeJWT failed to get correct userID")
	}

	wrongTokenSecret := "2945fesdff6-2470-4cc0-ab15-38608dfc74asdb8"

	if _, err := ValidateJWT(tokenString, wrongTokenSecret); !errors.Is(err, jwt.ErrSignatureInvalid) {
		t.Fatalf("validateJWT didn't return an invalid signature when given an invalid signature:  %s", err)
	}

	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIyOTQ1ZmVmNi0yNDcwLTRjYzAtYWIxNS0zODYwOGRmYzc0YjgiLCJleHAiOjE3NTMxNTMyMjIsImlhdCI6MTc1MzE1MzIyMX0.mBmIacO7SQhc9ovgeaRxUI7EXtVbD3HZiFs9vF7x1Lk"

	if _, err := ValidateJWT(expiredToken, tokenSecret); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("validateJWT didn't return expired for an expired token %s: ", err)
	}
}

func TestGetBearerToken(t *testing.T) {
	// check if it will return the token from a header

	// Mock http.Header for testing
	headers := http.Header{
		"Accept":            []string{"application/json", "text/plain", "*/*"},
		"Accept-Encoding":   []string{"gzip, deflate, br"},
		"Accept-Language":   []string{"en-US,en;q=0.9"},
		"Authorization":     []string{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
		"Cache-Control":     []string{"no-cache"},
		"Connection":        []string{"keep-alive"},
		"Content-Length":    []string{"1234"},
		"Content-Type":      []string{"application/json"},
		"Cookie":            []string{"session_id=abc123; user_pref=dark_mode"},
		"Host":              []string{"api.example.com"},
		"Origin":            []string{"https://example.com"},
		"Referer":           []string{"https://example.com/dashboard"},
		"User-Agent":        []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"},
		"X-Api-Key":         []string{"sk_test_1234567890abcdef"},
		"X-Client-Version":  []string{"1.2.3"},
		"X-Custom-Header":   []string{"value1", "value2"},
		"X-Device-Id":       []string{"device_abc123xyz789"},
		"X-Forwarded-For":   []string{"192.168.1.100, 10.0.0.1"},
		"X-Forwarded-Proto": []string{"https"},
		"X-Real-Ip":         []string{"192.168.1.100"},
		"X-Request-Id":      []string{"req_1234567890abcdef"},
	}

	if TokenString, err := GetBearerToken(headers); err != nil {
		t.Fatalf("GetBearerToken returned an error when it shouldn't have: %s", err)
	} else if TokenString == "" {
		t.Fatalf("GetBearerToken returned an empty string instead of the token")
	} else if TokenString != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" {
		t.Fatalf("GetBearerToken didn't return the correct token string, it returned this: %v", TokenString)
	}

	headers.Set("Authorization", "")

	if _, err := GetBearerToken(headers);err == nil || err.Error() != "failed to get autorization" {
		t.Fatalf("GetBearerToken didn't return the correct autorization failed error: %s", err)
	}

	headers.Set("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	if _, err := GetBearerToken(headers);err == nil || err.Error() != "incorrect autorization string format" {
		t.Fatalf("GetBearerToken didn't return the correct incorrect autorization string format error: %s", err)
	}

	headers.Set("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c asd adsf adsf")
	if _, err := GetBearerToken(headers); err == nil || err.Error() != "incorrect autorization string format" {
		t.Fatalf("GetBearerToken didn't return the correct incorrect autorization string format error: %s", err)
	}
}
