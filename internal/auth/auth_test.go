package auth

import (
	"testing"
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
