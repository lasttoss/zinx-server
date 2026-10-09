package utils

import (
	"encoding/json"
	"testing"
)

func TestNewApiError_IsValidJSONWithCodeAndMessage(t *testing.T) {
	for _, code := range []ErrorCode{InvalidRequestError, ItemNotFoundError, AnotherDeviceLoginError} {
		raw := NewApiError(code)

		var payload ApiError
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("NewApiError(%d) is not valid JSON: %v", code, err)
		}
		if payload.Code != int(code) {
			t.Fatalf("code = %d, want %d", payload.Code, code)
		}
		if payload.Message == "" {
			t.Fatalf("NewApiError(%d) has an empty message", code)
		}
	}
}
