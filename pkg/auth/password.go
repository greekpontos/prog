package auth

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmptyPassword    = errors.New("password cannot be empty")
	ErrPasswordTooShort = errors.New("password must be at least 6 characters long")
	ErrPasswordTooLong  = errors.New("password cannot exceed 72 bytes")
)

// PasswordCost задает вычислительную сложность алгоритма bcrypt
const PasswordCost = bcrypt.DefaultCost

// HashPassword хеширует переданный пароль с использованием алгоритма bcrypt
func HashPassword(password string) (string, error) {
	if err := ValidatePasswordStrength(password); err != nil {
		return "", err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), PasswordCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// CheckPassword проверяет соответствие открытого пароля и его bcrypt-хеша
func CheckPassword(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength проверяет требования к надежности пароля и ограничения bcrypt
func ValidatePasswordStrength(password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrEmptyPassword
	}
	if len([]rune(password)) < 6 {
		return ErrPasswordTooShort
	}
	// Ограничение алгоритма bcrypt — не более 72 байт входных данных
	if len([]byte(password)) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

// GenerateRandomPassword генерирует криптографически стойкий случайный пароль заданной длины
func GenerateRandomPassword(length int) (string, error) {
	if length <= 0 {
		length = 12
	}
	if length > 72 {
		length = 72
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	res := make([]byte, length)
	for i := range res {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		res[i] = charset[num.Int64()]
	}
	return string(res), nil
}
