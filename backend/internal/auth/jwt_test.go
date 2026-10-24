package auth

import (
	"testing"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret"
	userID := int64(42)

	token, err := GenerateToken(userID, secret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	gotUserID, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if gotUserID != userID {
		t.Errorf("Expected user ID %d, got %d", userID, gotUserID)
	}
}

func TestValidateTokenWithWrongSecret(t *testing.T) {
	token, err := GenerateToken(42, "correct-secret")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	_, err = ValidateToken(token, "wrong-secret")

	if err == nil {
		t.Error("Expected token validation to fail with wrong secret")
	}
}

func TestValidateMalformedToken(t *testing.T) {
	_, err := ValidateToken("not-a-valid-jwt", "test-secret")

	if err == nil {
		t.Error("Expected malformed token to be rejected")
	}
}
