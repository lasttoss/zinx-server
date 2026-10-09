package repositories

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"zinx-server/internal/constants"
	"zinx-server/internal/models"
)

// What is under test here is the query and the bson tags behind it: a wrong tag does not fail, it
// silently matches nothing, and the sign-in path turns "nothing" into "item not found". So these
// tests write a user and read it back through every finder.
//
// They need a real MongoDB and skip themselves without one, the same way the store tests in go-log-api
// need postgres:
//
//	docker compose up -d mongodb
//	TEST_MONGO_URI='mongodb://root:devpassword@localhost:27017/?authSource=admin' go test ./internal/repositories/
//
// The tests own the database they are pointed at: they drop the users collection. Do not point them
// at anything worth keeping.

const testDatabase = "zinx_server_test"

func open(t *testing.T) UserRepository {
	t.Helper()
	uri := strings.TrimSpace(os.Getenv("TEST_MONGO_URI"))
	if uri == "" {
		t.Skip("TEST_MONGO_URI is not set: these tests need a real MongoDB (see the comment above)")
	}

	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	drop(t, client)
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { drop(t, client) })

	return NewUserRepository(client.Database(testDatabase).Collection(constants.UserCollectionName))
}

func drop(t *testing.T, client *mongo.Client) {
	t.Helper()
	if err := client.Database(testDatabase).Collection(constants.UserCollectionName).Drop(context.Background()); err != nil {
		t.Fatalf("dropping the users collection: %v", err)
	}
}

// savedUser is one row with every identity set, so a finder that reads the wrong field cannot pass
// by accident.
func savedUser(t *testing.T, repo UserRepository) *models.UserModel {
	t.Helper()
	user := &models.UserModel{
		Id:          bson.NewObjectID(),
		UserId:      "user-1",
		DeviceId:    "device-1",
		GoogleId:    "google-1",
		AppleId:     "apple-1",
		FacebookId:  "facebook-1",
		DisplayName: "Dung",
		AvatarUrl:   "https://example.test/a.png",
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := repo.Save(user); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return user
}

func TestSave_StoresEveryIdentityUnderTheFieldTheFinderQueries(t *testing.T) {
	repo := open(t)
	user := savedUser(t, repo)

	tests := []struct {
		field string
		find  func(string) (*models.UserModel, error)
		key   string
	}{
		{"user_id", repo.FindByUserId, user.UserId},
		{"device_id", repo.FindByDeviceId, user.DeviceId},
		{"google_id", repo.FindByGoogleId, user.GoogleId},
		{"apple_id", repo.FindByAppleId, user.AppleId},
	}

	for _, test := range tests {
		t.Run(test.field, func(t *testing.T) {
			found, err := test.find(test.key)
			if err != nil {
				t.Fatalf("find by %s: %v", test.field, err)
			}
			if found == nil {
				t.Fatalf("nothing was found by %s: the bson tag or the query does not match what Save writes", test.field)
			}
			if found.Id != user.Id || found.UserId != user.UserId || found.DeviceId != user.DeviceId ||
				found.GoogleId != user.GoogleId || found.AppleId != user.AppleId ||
				found.FacebookId != user.FacebookId || found.DisplayName != user.DisplayName {
				t.Errorf("the stored row came back changed: %+v", found)
			}
			if !found.CreatedAt.Equal(user.CreatedAt) {
				t.Errorf("created_at = %s, want %s", found.CreatedAt, user.CreatedAt)
			}
		})
	}
}

// "No documents" is not an error for the callers of this package: the sign-in code asks whether an
// account exists, and a missing one is answered with a new account, not with a failure.
func TestFinders_AnswerNilRatherThanAnErrorWhenThereIsNoMatch(t *testing.T) {
	repo := open(t)
	savedUser(t, repo)

	tests := []struct {
		name string
		find func(string) (*models.UserModel, error)
	}{
		{"user_id", repo.FindByUserId},
		{"device_id", repo.FindByDeviceId},
		{"google_id", repo.FindByGoogleId},
		{"apple_id", repo.FindByAppleId},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			found, err := test.find("nobody-has-this")
			if err != nil {
				t.Fatalf("a missing row came back as an error: %v", err)
			}
			if found != nil {
				t.Errorf("found %+v for an id nobody has", found)
			}
		})
	}
}

func TestFindByUserId_DoesNotCrossIdentities(t *testing.T) {
	repo := open(t)
	user := savedUser(t, repo)

	// The device id of one user must not answer the question "who has this user id".
	found, err := repo.FindByUserId(user.DeviceId)
	if err != nil {
		t.Fatalf("FindByUserId: %v", err)
	}
	if found != nil {
		t.Errorf("FindByUserId matched a device id: %+v", found)
	}
}

// This is the call the sign-in paths make on every successful login, so it has to move the field it
// says it moves - and the row it updates is the one the caller read.
func TestUpdate_TouchesTheDocumentItIsGiven(t *testing.T) {
	repo := open(t)
	user := savedUser(t, repo)

	updatedAt := time.Now().UTC().Truncate(time.Millisecond)
	if err := repo.Update(user.Id, bson.M{"$set": bson.M{"updated_at": updatedAt, "display_name": "Renamed"}}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := repo.FindByUserId(user.UserId)
	if err != nil {
		t.Fatalf("FindByUserId: %v", err)
	}
	if found == nil {
		t.Fatal("the user disappeared after being updated")
	}
	if !found.UpdatedAt.Equal(updatedAt) {
		t.Errorf("updated_at = %s, want %s", found.UpdatedAt, updatedAt)
	}
	if found.DisplayName != "Renamed" {
		t.Errorf("display_name = %q, want Renamed", found.DisplayName)
	}
	if found.CreatedAt != user.CreatedAt {
		t.Errorf("created_at moved: %s, want %s", found.CreatedAt, user.CreatedAt)
	}
}

// An update that matches nothing is not an error, so a caller holding a stale document id gets no
// signal at all. Recorded because the sign-in code treats an error from Update as fatal, and this is
// the case where it will hear nothing.
func TestUpdate_IsSilentAboutAnIdThatIsNotThere(t *testing.T) {
	repo := open(t)
	savedUser(t, repo)

	if err := repo.Update(bson.NewObjectID(), bson.M{"$set": bson.M{"display_name": "Nobody"}}); err != nil {
		t.Errorf("updating a document that does not exist returned %v, want nil", err)
	}
}

// Save inserts, so it fails on a second call with the same document. Worth pinning down: the
// provider paths call Save for a user they have just built, and a retry would be an error rather
// than an update.
func TestSave_FailsOnADocumentThatIsAlreadyThere(t *testing.T) {
	repo := open(t)
	user := savedUser(t, repo)

	if err := repo.Save(user); err == nil {
		t.Error("Save accepted a second insert of the same document: it must be an insert, not an upsert")
	}
}
