package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"zinx-server/internal/configs"
	"zinx-server/internal/constants"
	"zinx-server/internal/mappers"
	"zinx-server/internal/models"
	"zinx-server/internal/utils"
)

// The auth service is the first thing a client talks to and the only place that decides who a
// player is, so these tests pin down the answer for every branch: which error code, whether the
// repository was asked to save or to update, and what the response body contains.
//
// Everything here goes through the UserRepository interface, so no MongoDB is involved. The
// repository's own SQL - the queries and the "no documents" to nil translation - is in
// repositories, and is tested there against a real database.

const (
	googleClientID      = "google-client-id"
	appleGoogleClientID = "apple-google-client-id"
	appleClientID       = "apple-client-id"
)

type fakeUserRepository struct {
	byDevice *models.UserModel
	byUserID *models.UserModel
	byGoogle *models.UserModel
	byApple  *models.UserModel
	shared   *models.UserModel // returned in place of any of the above when set

	findErr   error
	saveErr   error
	updateErr error

	saved   []*models.UserModel
	updated []bson.ObjectID
}

func (f *fakeUserRepository) FindByDeviceId(string) (*models.UserModel, error) {
	return f.pick(f.byDevice), f.findErr
}

func (f *fakeUserRepository) FindByUserId(string) (*models.UserModel, error) {
	return f.pick(f.byUserID), f.findErr
}

func (f *fakeUserRepository) FindByGoogleId(string) (*models.UserModel, error) {
	return f.pick(f.byGoogle), f.findErr
}

func (f *fakeUserRepository) FindByAppleId(string) (*models.UserModel, error) {
	return f.pick(f.byApple), f.findErr
}

func (f *fakeUserRepository) pick(specific *models.UserModel) *models.UserModel {
	if f.shared != nil {
		return f.shared
	}
	return specific
}

func (f *fakeUserRepository) Save(user *models.UserModel) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, user)
	return nil
}

func (f *fakeUserRepository) Update(id bson.ObjectID, _ bson.M) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = append(f.updated, id)
	return nil
}

func withJwtSecret(t *testing.T) {
	t.Helper()
	previous := configs.ServerConfig.Jwt.Secret
	configs.ServerConfig.Jwt.Secret = "auth-service-test-secret"
	t.Cleanup(func() { configs.ServerConfig.Jwt.Secret = previous })
}

func newAuthService(repo *fakeUserRepository) AuthService {
	return NewAuthService(repo, googleClientID, appleGoogleClientID, appleClientID)
}

// signedToken builds what a provider or this service would hand over. ParseUnverified is what the
// Google and Apple paths use, so a token from "a provider" only has to be a well-formed JWT.
func signedToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(configs.ServerConfig.Jwt.Secret))
	if err != nil {
		t.Fatalf("signing the test token: %v", err)
	}
	return signed
}

func assertErrorCode(t *testing.T, errBytes []byte, code utils.ErrorCode) {
	t.Helper()
	if want := string(utils.NewApiError(code)); string(errBytes) != want {
		t.Errorf("error = %s, want %s", errBytes, want)
	}
}

func assertAuthResponse(t *testing.T, userID string, body []byte) mappers.AuthResponse {
	t.Helper()
	var response mappers.AuthResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("the response is not the shape a client expects (%v): %s", err, body)
	}
	if response.Token == "" {
		t.Error("the response carries no token")
	}
	if response.Data.UserId != userID {
		t.Errorf("response user = %q, want %q", response.Data.UserId, userID)
	}
	return response
}

func TestAuthByToken_AcceptsItsOwnTokenForAKnownUser(t *testing.T) {
	withJwtSecret(t)

	existing := models.NewDeviceIdUserModel("device-1")
	savedUpdatedAt := existing.UpdatedAt
	repo := &fakeUserRepository{byUserID: existing}
	service := newAuthService(repo)

	token := signedToken(t, jwt.MapClaims{"sub": existing.UserId, "exp": time.Now().Add(time.Hour).Unix()})
	userID, body, errBytes := service.AuthByToken(mappers.AuthRequest{Id: token})

	if errBytes != nil {
		t.Fatalf("unexpected error: %s", errBytes)
	}
	if userID != existing.UserId {
		t.Errorf("user id = %q, want %q", userID, existing.UserId)
	}
	assertAuthResponse(t, existing.UserId, body)

	// The row is touched so that "last seen" has a meaning, and it is the row that was read.
	if len(repo.updated) != 1 || repo.updated[0] != existing.Id {
		t.Errorf("updated = %v, want exactly the user that signed in (%s)", repo.updated, existing.Id)
	}
	if !existing.UpdatedAt.After(savedUpdatedAt) {
		t.Error("updated_at was not moved forward")
	}
	if len(repo.saved) != 0 {
		t.Errorf("Save was called %d times for an existing user", len(repo.saved))
	}
}

func TestAuthByToken_RejectsTokensItCannotTrust(t *testing.T) {
	withJwtSecret(t)

	tests := []struct {
		name  string
		token string
	}{
		{name: "garbage", token: "not-a-jwt"},
		{
			name:  "signed with another secret",
			token: mustSign(t, "a-different-secret", jwt.MapClaims{"sub": "someone", "exp": time.Now().Add(time.Hour).Unix()}),
		},
		{
			name:  "expired",
			token: signedToken(t, jwt.MapClaims{"sub": "someone", "exp": time.Now().Add(-time.Hour).Unix()}),
		},
		{
			name:  "without a subject",
			token: signedToken(t, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeUserRepository{shared: models.NewDeviceIdUserModel("device-1")}
			if _, body, errBytes := newAuthService(repo).AuthByToken(mappers.AuthRequest{Id: test.token}); errBytes == nil {
				t.Fatalf("the service answered with %s", body)
			} else {
				assertErrorCode(t, errBytes, utils.JwtClaimsError)
			}
		})
	}
}

func TestAuthByToken_ReportsARepositoryFailureAsASystemError(t *testing.T) {
	withJwtSecret(t)
	user := models.NewDeviceIdUserModel("device-1")

	t.Run("the lookup fails", func(t *testing.T) {
		repo := &fakeUserRepository{shared: user, findErr: errors.New("mongo is down")}
		token := signedToken(t, jwt.MapClaims{"sub": user.UserId, "exp": time.Now().Add(time.Hour).Unix()})
		_, body, errBytes := newAuthService(repo).AuthByToken(mappers.AuthRequest{Id: token})

		if body != nil {
			t.Errorf("a failed lookup still produced a body: %s", body)
		}
		assertErrorCode(t, errBytes, utils.SystemError)
	})

	t.Run("the update fails", func(t *testing.T) {
		repo := &fakeUserRepository{shared: user, updateErr: errors.New("mongo is down")}
		token := signedToken(t, jwt.MapClaims{"sub": user.UserId, "exp": time.Now().Add(time.Hour).Unix()})
		_, _, errBytes := newAuthService(repo).AuthByToken(mappers.AuthRequest{Id: token})

		assertErrorCode(t, errBytes, utils.SystemError)
	})
}

// A token that is valid but names a user that no longer exists is a 404-shaped answer for the
// client, not a system error: "log in again" and "come back later" are different instructions.
func TestAuthByToken_ReportsAUserThatIsGone(t *testing.T) {
	withJwtSecret(t)

	repo := &fakeUserRepository{}
	token := signedToken(t, jwt.MapClaims{"sub": "deleted-user", "exp": time.Now().Add(time.Hour).Unix()})
	_, _, errBytes := newAuthService(repo).AuthByToken(mappers.AuthRequest{Id: token})

	assertErrorCode(t, errBytes, utils.ItemNotFoundError)
}

func mustSign(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing with %q: %v", secret, err)
	}
	return signed
}

func TestAuthByDevice_RegistersADeviceNobodyHasSeen(t *testing.T) {
	withJwtSecret(t)

	repo := &fakeUserRepository{}
	userID, body, errBytes := newAuthService(repo).AuthByDevice(mappers.AuthRequest{Id: "device-42"})

	if errBytes != nil {
		t.Fatalf("unexpected error: %s", errBytes)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("Save was called %d times, want 1", len(repo.saved))
	}
	if repo.saved[0].DeviceId != "device-42" {
		t.Errorf("saved device = %q", repo.saved[0].DeviceId)
	}
	if repo.saved[0].UserId == "" {
		t.Error("the new user has no user id, so nothing can look it up later")
	}
	if userID != repo.saved[0].UserId {
		t.Errorf("returned user id %q, saved %q", userID, repo.saved[0].UserId)
	}
	assertAuthResponse(t, repo.saved[0].UserId, body)
}

func TestAuthByDevice_TouchesADeviceItAlreadyKnows(t *testing.T) {
	withJwtSecret(t)

	existing := models.NewDeviceIdUserModel("device-42")
	repo := &fakeUserRepository{byDevice: existing}
	userID, body, errBytes := newAuthService(repo).AuthByDevice(mappers.AuthRequest{Id: "device-42"})

	if errBytes != nil {
		t.Fatalf("unexpected error: %s", errBytes)
	}
	if len(repo.saved) != 0 {
		t.Errorf("Save was called %d times for a device that is already registered", len(repo.saved))
	}
	if len(repo.updated) != 1 || repo.updated[0] != existing.Id {
		t.Errorf("updated = %v, want the existing row %s", repo.updated, existing.Id)
	}
	if userID != existing.UserId {
		t.Errorf("user id = %q, want %q", userID, existing.UserId)
	}
	assertAuthResponse(t, existing.UserId, body)
}

func TestAuthByDevice_ReportsRepositoryFailures(t *testing.T) {
	withJwtSecret(t)

	tests := []struct {
		name string
		repo *fakeUserRepository
	}{
		{name: "the lookup fails", repo: &fakeUserRepository{findErr: errors.New("mongo is down")}},
		{name: "the insert fails", repo: &fakeUserRepository{saveErr: errors.New("mongo is down")}},
		{
			name: "the update fails",
			repo: &fakeUserRepository{byDevice: models.NewDeviceIdUserModel("device-42"), updateErr: errors.New("mongo is down")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, errBytes := newAuthService(test.repo).AuthByDevice(mappers.AuthRequest{Id: "device-42"})
			assertErrorCode(t, errBytes, utils.SystemError)
		})
	}
}

// A provider token is only accepted for the client ids this service was configured with. The two
// rejections below are the point of the check: a token minted for somebody else's app must not
// become a login here.
func TestAuthByGoogle_OnlyAcceptsItsOwnAudience(t *testing.T) {
	withJwtSecret(t)

	tests := []struct {
		name    string
		aud     any
		wantErr bool
	}{
		{name: "the android/web client id", aud: googleClientID},
		{name: "the apple-google client id", aud: appleGoogleClientID},
		{name: "another application's client id", aud: "somebody-elses-client-id", wantErr: true},
		{name: "not a string", aud: 12345, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			claims := jwt.MapClaims{constants.JwtSub: "google-1", "exp": time.Now().Add(time.Hour).Unix()}
			if test.aud != nil {
				claims[constants.JwtAud] = test.aud
			}
			_, body, errBytes := newAuthService(repo).AuthByGoogle(mappers.AuthRequest{Id: signedToken(t, claims)})

			if test.wantErr {
				if errBytes == nil {
					t.Fatalf("a token for %v was accepted: %s", test.aud, body)
				}
				assertErrorCode(t, errBytes, utils.JwtClaimsError)
				if len(repo.saved) != 0 {
					t.Error("a rejected token still created a user")
				}
				return
			}
			if errBytes != nil {
				t.Fatalf("unexpected error: %s", errBytes)
			}
			if len(repo.saved) != 1 || repo.saved[0].GoogleId != "google-1" {
				t.Errorf("saved = %+v, want one user with google_id google-1", repo.saved)
			}
		})
	}
}

func TestAuthByGoogle_TouchesAUserItAlreadyKnows(t *testing.T) {
	withJwtSecret(t)

	existing := models.NewGoogleIdUserModel("google-1")
	existing.UserId = "user-of-google-1"
	repo := &fakeUserRepository{byGoogle: existing}
	claims := jwt.MapClaims{
		constants.JwtSub: "google-1",
		constants.JwtAud: googleClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
	}
	userID, body, errBytes := newAuthService(repo).AuthByGoogle(mappers.AuthRequest{Id: signedToken(t, claims)})

	if errBytes != nil {
		t.Fatalf("unexpected error: %s", errBytes)
	}
	if len(repo.saved) != 0 {
		t.Errorf("Save was called %d times for a known google id", len(repo.saved))
	}
	if len(repo.updated) != 1 || repo.updated[0] != existing.Id {
		t.Errorf("updated = %v, want %s", repo.updated, existing.Id)
	}
	assertAuthResponse(t, userID, body)
}

func TestAuthByGoogle_ReportsRepositoryFailures(t *testing.T) {
	withJwtSecret(t)
	claims := jwt.MapClaims{
		constants.JwtSub: "google-1",
		constants.JwtAud: googleClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
	}

	tests := []struct {
		name string
		repo *fakeUserRepository
	}{
		{name: "the lookup fails", repo: &fakeUserRepository{findErr: errors.New("mongo is down")}},
		{name: "the insert fails", repo: &fakeUserRepository{saveErr: errors.New("mongo is down")}},
		{
			name: "the update fails",
			repo: &fakeUserRepository{byGoogle: models.NewGoogleIdUserModel("google-1"), updateErr: errors.New("mongo is down")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, errBytes := newAuthService(test.repo).AuthByGoogle(mappers.AuthRequest{Id: signedToken(t, claims)})
			assertErrorCode(t, errBytes, utils.SystemError)
		})
	}
}

func TestAuthByGoogle_RejectsATokenWithoutASubject(t *testing.T) {
	withJwtSecret(t)

	claims := jwt.MapClaims{constants.JwtAud: googleClientID, "exp": time.Now().Add(time.Hour).Unix()}
	_, _, errBytes := newAuthService(&fakeUserRepository{}).AuthByGoogle(mappers.AuthRequest{Id: signedToken(t, claims)})

	assertErrorCode(t, errBytes, utils.JwtClaimsError)
}

func TestAuthByApple_OnlyAcceptsItsOwnAudience(t *testing.T) {
	withJwtSecret(t)

	tests := []struct {
		name    string
		aud     string
		wantErr bool
	}{
		{name: "the apple client id", aud: appleClientID},
		{name: "the google client id", aud: googleClientID, wantErr: true},
		{name: "another application's client id", aud: "somebody-elses-client-id", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			claims := jwt.MapClaims{
				constants.JwtSub: "apple-1",
				constants.JwtAud: test.aud,
				"exp":            time.Now().Add(time.Hour).Unix(),
			}
			_, body, errBytes := newAuthService(repo).AuthByApple(mappers.AuthRequest{Id: signedToken(t, claims)})

			if test.wantErr {
				if errBytes == nil {
					t.Fatalf("a token for %q was accepted: %s", test.aud, body)
				}
				assertErrorCode(t, errBytes, utils.JwtClaimsError)
				return
			}
			if errBytes != nil {
				t.Fatalf("unexpected error: %s", errBytes)
			}
			if len(repo.saved) != 1 || repo.saved[0].AppleId != "apple-1" {
				t.Errorf("saved = %+v, want one user with apple_id apple-1", repo.saved)
			}
		})
	}
}

func TestAuthByApple_TouchesAUserItAlreadyKnows(t *testing.T) {
	withJwtSecret(t)

	existing := models.NewAppleIdUserModel("apple-1")
	existing.UserId = "user-of-apple-1"
	repo := &fakeUserRepository{byApple: existing}
	claims := jwt.MapClaims{
		constants.JwtSub: "apple-1",
		constants.JwtAud: appleClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
	}
	userID, body, errBytes := newAuthService(repo).AuthByApple(mappers.AuthRequest{Id: signedToken(t, claims)})

	if errBytes != nil {
		t.Fatalf("unexpected error: %s", errBytes)
	}
	if len(repo.saved) != 0 {
		t.Errorf("Save was called %d times for a known apple id", len(repo.saved))
	}
	if len(repo.updated) != 1 || repo.updated[0] != existing.Id {
		t.Errorf("updated = %v, want %s", repo.updated, existing.Id)
	}
	assertAuthResponse(t, userID, body)
}

func TestAuthByApple_ReportsRepositoryFailures(t *testing.T) {
	withJwtSecret(t)
	claims := jwt.MapClaims{
		constants.JwtSub: "apple-1",
		constants.JwtAud: appleClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
	}

	tests := []struct {
		name string
		repo *fakeUserRepository
	}{
		{name: "the lookup fails", repo: &fakeUserRepository{findErr: errors.New("mongo is down")}},
		{name: "the insert fails", repo: &fakeUserRepository{saveErr: errors.New("mongo is down")}},
		{
			name: "the update fails",
			repo: &fakeUserRepository{byApple: models.NewAppleIdUserModel("apple-1"), updateErr: errors.New("mongo is down")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, errBytes := newAuthService(test.repo).AuthByApple(mappers.AuthRequest{Id: signedToken(t, claims)})
			assertErrorCode(t, errBytes, utils.SystemError)
		})
	}
}

// This is what a player's account looks like on the day it is created through a provider. It is
// recorded here because it is surprising: unlike the device path, the provider constructors leave
// user_id empty, so the token this service issues has an empty subject and nothing can look the
// account up by user id afterwards. Flagged rather than changed - see the note in the commit.
func TestAuthByGoogle_ANewProviderAccountHasNoUserIdYet(t *testing.T) {
	withJwtSecret(t)

	repo := &fakeUserRepository{}
	claims := jwt.MapClaims{
		constants.JwtSub: "google-1",
		constants.JwtAud: googleClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
	}
	userID, body, errBytes := newAuthService(repo).AuthByGoogle(mappers.AuthRequest{Id: signedToken(t, claims)})

	if errBytes != nil {
		t.Fatalf("unexpected error: %s", errBytes)
	}
	if userID != "" {
		t.Skip(fmt.Sprintf("user_id is now %q - models.NewGoogleIdUserModel was fixed, extend the response checks here", userID))
	}
	if response := assertAuthResponse(t, "", body); response.Data.Id == "" {
		t.Error("the response has neither a user id nor a document id, so a client cannot identify the account")
	}
}
