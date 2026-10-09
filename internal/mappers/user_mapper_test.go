package mappers

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/v2/bson"
	"testing"
	"time"

	"zinx-server/internal/models"
)

// The mapper is the only place that decides what a client is allowed to see, so these tests check
// both halves: every field arrives, and the JSON names are the ones a client already expects - a
// rename here is a breaking change for every released game.

func TestNewUserResponse_CarriesEveryField(t *testing.T) {
	created := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	item := models.UserModel{
		Id:          bson.NewObjectID(),
		UserId:      "user-1",
		GoogleId:    "google-1",
		AppleId:     "apple-1",
		FacebookId:  "facebook-1",
		DeviceId:    "device-1",
		DisplayName: "Dung",
		AvatarUrl:   "https://example.test/a.png",
		CreatedAt:   created,
		UpdatedAt:   created.Add(time.Hour),
	}

	got := NewUserResponse(item)

	if got.Id != item.Id.Hex() {
		t.Errorf("id = %q, want the object id in hex (%q)", got.Id, item.Id.Hex())
	}
	if got.UserId != item.UserId || got.GoogleId != item.GoogleId || got.AppleId != item.AppleId ||
		got.FacebookId != item.FacebookId || got.DeviceId != item.DeviceId ||
		got.DisplayName != item.DisplayName || got.AvatarUrl != item.AvatarUrl {
		t.Errorf("a field was dropped: %+v", got)
	}
	if got.CreatedAt != item.CreatedAt.String() || got.UpdatedAt != item.UpdatedAt.String() {
		t.Errorf("timestamps = %q/%q, want %q/%q", got.CreatedAt, got.UpdatedAt, item.CreatedAt, item.UpdatedAt)
	}
}

func TestNewUserResponse_KeepsTheJsonNamesClientsDependOn(t *testing.T) {
	body, err := json.Marshal(NewUserResponse(*models.NewDeviceIdUserModel("device-1")))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{
		"id", "userId", "deviceId", "googleId", "facebookId", "appleId",
		"displayName", "avatarUrl", "createdAt", "updatedAt",
	} {
		if _, ok := decoded[field]; !ok {
			t.Errorf("the response has no %q field: %s", field, body)
		}
	}
}

func TestNewAuthResponse_PairsTheTokenWithTheUser(t *testing.T) {
	user := NewUserResponse(*models.NewDeviceIdUserModel("device-1"))
	response := NewAuthResponse("signed-token", user)

	if response.Token != "signed-token" {
		t.Errorf("token = %q", response.Token)
	}
	if response.Data.UserId != user.UserId {
		t.Errorf("user = %q, want %q", response.Data.UserId, user.UserId)
	}
}
