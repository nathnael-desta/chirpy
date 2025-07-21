package auth

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	newPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(newPassword), err
}

func CheckPasswordHash(password, hash string) error {
	return  bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}