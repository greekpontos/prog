package auth

import "testing"

func TestPasswordHashing(t *testing.T) {
	password := "123456789"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hash == password {
		t.Fatal("hash must not match plain password")
	}
	if !CheckPassword(password, hash) {
		t.Fatal("expected password to match hash")
	}
	if CheckPassword("wrongpass", hash) {
		t.Fatal("expected wrong password not to match")
	}
}

func TestJWTManager(t *testing.T) {
	jm := NewJWTManager("test-secret-key", 1)
	token, expiresAt, err := jm.GenerateToken(1, "greekpotnos@mail.ru", "greek")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token == "" || expiresAt.IsZero() {
		t.Fatal("expected valid token and expiration time")
	}

	claims, err := jm.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if claims.UserID != 1 || claims.Email != "greekpotnos@mail.ru" || claims.Username != "greek" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
