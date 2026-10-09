package utils

import (
	"testing"
	"time"

	"zinx-server/internal/configs"
	"zinx-server/internal/models"
)

func withSecret(t *testing.T) {
	t.Helper()
	previous := configs.ServerConfig.Jwt.Secret
	configs.ServerConfig.Jwt.Secret = "unit-test-secret"
	t.Cleanup(func() { configs.ServerConfig.Jwt.Secret = previous })
}

func TestGenerateJWT_ContainsSubjectAndExpiry(t *testing.T) {
	withSecret(t)

	user := models.NewDeviceIdUserModel("device-1")
	token, err := GenerateJWT(user, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}

	if got := claims["sub"]; got != user.UserId {
		t.Fatalf("sub = %v, want %v", got, user.UserId)
	}
	if exp, ok := claims["exp"].(float64); !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		t.Fatalf("exp is missing or in the past: %v", claims["exp"])
	}
}

func TestValidateJWT_RejectsExpiredToken(t *testing.T) {
	withSecret(t)

	user := models.NewDeviceIdUserModel("device-1")
	token, err := GenerateJWT(user, -time.Minute)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	if _, err := ValidateJWT(token); err == nil {
		t.Fatal("ValidateJWT() accepted an expired token")
	}
}

func TestValidateJWT_RejectsWrongSecret(t *testing.T) {
	withSecret(t)

	user := models.NewDeviceIdUserModel("device-1")
	token, err := GenerateJWT(user, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	configs.ServerConfig.Jwt.Secret = "another-secret"
	if _, err := ValidateJWT(token); err == nil {
		t.Fatal("ValidateJWT() accepted a token signed with another secret")
	}
}

func TestValidateJWT_RejectsGarbage(t *testing.T) {
	withSecret(t)

	if _, err := ValidateJWT("not-a-token"); err == nil {
		t.Fatal("ValidateJWT() accepted garbage")
	}
}
