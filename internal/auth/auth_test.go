package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	gotID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}

	if gotID != userID {
		t.Errorf("expected %v, got %v", userID, gotID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Error("expected error with wrong secret, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, "secret", -time.Second)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	_, err = ValidateJWT(token, "secret")
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	_, err := ValidateJWT("invalid.token.string", "secret")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer mytoken123")

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "mytoken123" {
		t.Errorf("expected mytoken123, got %v", token)
	}
}

func TestGetBearerToken_EmptyHeader(t *testing.T) {
	_, err := GetBearerToken(http.Header{})
	if err == nil {
		t.Error("expected error for empty header, got nil")
	}
}

func TestGetBearerToken_NoBearer(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "mytoken123")

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error("expected error when Bearer prefix is missing, got nil")
	}
}
