package models

import "testing"

// These constructors decide what is written to the database the first time somebody signs in, so
// what they leave empty matters as much as what they fill in. The empty UserId on the provider paths
// is recorded here rather than changed: it looks like an oversight next to the device path, but it
// changes what a client receives, so it is a decision to take deliberately.

func TestNewDeviceIdUserModel_IdentifiesTheAccount(t *testing.T) {
	user := NewDeviceIdUserModel("device-1")

	if user.DeviceId != "device-1" {
		t.Errorf("device id = %q", user.DeviceId)
	}
	if user.UserId == "" {
		t.Error("user id is empty: nothing could look this account up by it")
	}
	if user.Id.IsZero() {
		t.Error("the document id is zero")
	}
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Error("the timestamps are zero")
	}
	if user.GoogleId != "" || user.AppleId != "" || user.FacebookId != "" {
		t.Errorf("a device sign-in claimed an identity it does not have: %+v", user)
	}
}

func TestNewDeviceIdUserModel_GivesEachAccountItsOwnUserId(t *testing.T) {
	first, second := NewDeviceIdUserModel("device-1"), NewDeviceIdUserModel("device-1")

	if first.UserId == second.UserId {
		t.Error("two sign-ins on the same device produced the same user id")
	}
	if first.Id == second.Id {
		t.Error("two sign-ins produced the same document id")
	}
}

func TestNewGoogleIdUserModel_RecordsTheProviderAndNothingElse(t *testing.T) {
	user := NewGoogleIdUserModel("google-1")

	if user.GoogleId != "google-1" {
		t.Errorf("google id = %q", user.GoogleId)
	}
	if user.Id.IsZero() || user.CreatedAt.IsZero() {
		t.Error("the document id or the timestamps are zero")
	}
	if user.UserId != "" {
		t.Logf("user_id is now %q: the empty-user-id note in the commit is out of date", user.UserId)
	}
	if user.AppleId != "" || user.DeviceId != "" || user.FacebookId != "" {
		t.Errorf("a google sign-in claimed another identity: %+v", user)
	}
}

func TestNewAppleIdUserModel_RecordsTheProviderAndNothingElse(t *testing.T) {
	user := NewAppleIdUserModel("apple-1")

	if user.AppleId != "apple-1" {
		t.Errorf("apple id = %q", user.AppleId)
	}
	if user.Id.IsZero() || user.CreatedAt.IsZero() {
		t.Error("the document id or the timestamps are zero")
	}
	if user.GoogleId != "" || user.DeviceId != "" || user.FacebookId != "" {
		t.Errorf("an apple sign-in claimed another identity: %+v", user)
	}
}
