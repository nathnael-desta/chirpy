package auth

import (
	"errors"
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
	tokenString, err := makeJWT(userId, tokenSecret, time.Second)
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
	tokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIyOTQ1ZmVmNi0yNDcwLTRjYzAtYWIxNS0zODYwOGRmYzc0YjgiLCJleHAiOjE3NTMyMzkwNjQsImlhdCI6MTc1MzE1MjY2NH0.NW7l0Aj7sla6GnekG6Xg40Eq7CfQksXGjgB1lbzPvMA"
	tokenSecret := "f1d9cffa3564f3d1e75027ec2805382a"

	userID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Fatalf("ValidateJWT returned an error: %s ", err)
	}

	userIdAnswer, err := uuid.Parse("2945fef6-2470-4cc0-ab15-38608dfc74b8")

	if err != nil {
		t.Fatalf("failed to make uuid: %s", err)
	}

	if userID != userIdAnswer {
		t.Fatalf("makeJWT failed to get correct userID")
	}

	
	wrongTokenSecret := "2945fesdff6-2470-4cc0-ab15-38608dfc74asdb8"

	if _, err := ValidateJWT(tokenString, wrongTokenSecret) ; !errors.Is(err,jwt.ErrSignatureInvalid) {
		t.Fatalf("validateJWT didn't return an invalid signature when given an invalid signature:  %s", err)
	}

	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIyOTQ1ZmVmNi0yNDcwLTRjYzAtYWIxNS0zODYwOGRmYzc0YjgiLCJleHAiOjE3NTMxNTMyMjIsImlhdCI6MTc1MzE1MzIyMX0.mBmIacO7SQhc9ovgeaRxUI7EXtVbD3HZiFs9vF7x1Lk"

	if _, err := ValidateJWT(expiredToken, tokenSecret) ; !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("validateJWT didn't return expired for an expired token %s: ", err)
	}
}
