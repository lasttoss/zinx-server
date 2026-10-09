package handlers

import (
	"errors"
	"testing"

	"github.com/aceld/zinx/ziface"

	"zinx-server/internal/constants"
	"zinx-server/internal/mappers"
	"zinx-server/internal/services"
	"zinx-server/internal/utils"
)

// A router is the boundary between the socket and the service layer, and what goes wrong here is
// protocol-shaped: the wrong message id, a second message where a client expects one, or a session
// saved for a sign-in that failed. So these tests watch the connection itself.
//
// The zinx interfaces (IRequest, IConnection) are embedded rather than implemented - the methods a
// router actually calls are the ones overridden below, and anything else panicking is the signal
// that a test has wandered into zinx's own machinery.

type sentMessage struct {
	msgID uint32
	data  []byte
}

type fakeConnection struct {
	ziface.IConnection

	connID  uint64
	sent    []sentMessage
	props   map[string]any
	failFor *uint32 // SendMsg returns an error for this message id
}

func newFakeConnection(connID uint64) *fakeConnection {
	return &fakeConnection{connID: connID, props: map[string]any{}}
}

func (c *fakeConnection) SendMsg(msgID uint32, data []byte) error {
	if c.failFor != nil && *c.failFor == msgID {
		return errors.New("the write failed")
	}
	c.sent = append(c.sent, sentMessage{msgID: msgID, data: data})
	return nil
}

func (c *fakeConnection) GetConnID() uint64 { return c.connID }

func (c *fakeConnection) SetProperty(key string, value any) { c.props[key] = value }

func (c *fakeConnection) GetProperty(key string) (any, error) {
	value, ok := c.props[key]
	if !ok {
		return nil, errors.New("no such property")
	}
	return value, nil
}

type fakeRequest struct {
	ziface.IRequest

	conn  ziface.IConnection
	data  []byte
	msgID uint32
}

func (r *fakeRequest) GetConnection() ziface.IConnection { return r.conn }
func (r *fakeRequest) GetData() []byte                   { return r.data }
func (r *fakeRequest) GetMsgID() uint32                  { return r.msgID }

// authStub answers like the real service, and remembers that it was asked.
type authStub struct {
	userID   string
	response []byte
	err      []byte

	calls []string
}

func (f *authStub) answer(name string) (string, []byte, []byte) {
	f.calls = append(f.calls, name)
	return f.userID, f.response, f.err
}

func (f *authStub) AuthByToken(mappers.AuthRequest) (string, []byte, []byte) {
	return f.answer("token")
}

func (f *authStub) AuthByDevice(mappers.AuthRequest) (string, []byte, []byte) {
	return f.answer("device")
}

func (f *authStub) AuthByGoogle(mappers.AuthRequest) (string, []byte, []byte) {
	return f.answer("google")
}

func (f *authStub) AuthByApple(mappers.AuthRequest) (string, []byte, []byte) {
	return f.answer("apple")
}

type fakeRedisService struct {
	sessions [][2]any
}

func (f *fakeRedisService) ClearAllSessions() {}

func (f *fakeRedisService) SaveSession(userID string, sessionID uint64) {
	f.sessions = append(f.sessions, [2]any{userID, sessionID})
}

func (f *fakeRedisService) GetSession(string) (uint64, bool) { return 0, false }

func assertSingleMessage(t *testing.T, conn *fakeConnection, wantID uint32, wantBody []byte) {
	t.Helper()
	if len(conn.sent) != 1 {
		t.Fatalf("the client got %d messages, want exactly 1: %+v", len(conn.sent), conn.sent)
	}
	if conn.sent[0].msgID != wantID {
		t.Errorf("message id = %d, want %d", conn.sent[0].msgID, wantID)
	}
	if string(conn.sent[0].data) != string(wantBody) {
		t.Errorf("body = %s, want %s", conn.sent[0].data, wantBody)
	}
}

func TestAuthRouters_AnswerWithTheServiceResponse(t *testing.T) {
	const userID = "user-1"

	tests := []struct {
		name string
		// build returns the router's Handle with the fakes wired in, the way route.go wires them.
		build func(services.AuthService, services.RedisService) func(ziface.IRequest)
	}{
		{"auth by token", func(a services.AuthService, r services.RedisService) func(ziface.IRequest) {
			return (&AuthTokenRouter{AuthService: a, RedisService: r}).Handle
		}},
		{"auth by device", func(a services.AuthService, r services.RedisService) func(ziface.IRequest) {
			return (&AuthDeviceRouter{AuthService: a, RedisService: r}).Handle
		}},
		{"auth by google", func(a services.AuthService, r services.RedisService) func(ziface.IRequest) {
			return (&AuthGoogleRouter{AuthService: a, RedisService: r}).Handle
		}},
		{"auth by apple", func(a services.AuthService, r services.RedisService) func(ziface.IRequest) {
			return (&AuthAppleRouter{AuthService: a, RedisService: r}).Handle
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := newFakeConnection(7)
			redis := &fakeRedisService{}
			auth := &authStub{userID: userID, response: []byte(`{"token":"t"}`)}

			test.build(auth, redis)(&fakeRequest{conn: conn, data: []byte(`{"id":"x"}`), msgID: constants.RpcAuthByToken})

			assertSingleMessage(t, conn, constants.RpcAuthByToken, []byte(`{"token":"t"}`))
			if conn.props["userId"] != userID {
				t.Errorf("userId property = %v, want %q", conn.props["userId"], userID)
			}
			if len(redis.sessions) != 1 || redis.sessions[0][0] != userID || redis.sessions[0][1] != uint64(7) {
				t.Errorf("sessions = %+v, want one session for %s on connection 7", redis.sessions, userID)
			}
		})
	}
}

func TestAuthRouters_RejectABodyTheyCannotRead(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"not json", `{"id":`},
		{"missing the id the validator requires", `{}`},
	}

	build := func(a services.AuthService, r services.RedisService) func(ziface.IRequest) {
		return (&AuthTokenRouter{AuthService: a, RedisService: r}).Handle
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := newFakeConnection(7)
			redis := &fakeRedisService{}
			auth := &authStub{}

			build(auth, redis)(&fakeRequest{conn: conn, data: []byte(test.body), msgID: constants.RpcAuthByToken})

			assertSingleMessage(t, conn, constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
			if len(auth.calls) != 0 {
				t.Errorf("the service was asked to sign somebody in anyway: %v", auth.calls)
			}
			if len(redis.sessions) != 0 {
				t.Errorf("a rejected request stored a session: %+v", redis.sessions)
			}
		})
	}
}

// A sign-in that failed must answer once and stop. The second message is not just noise: it is a
// response to a request that was never answered, and it registers a session for a user id that the
// failed call never produced.
func TestAuthRouters_AnswerOnceWhenTheSignInFails(t *testing.T) {
	serviceError := utils.NewApiError(utils.SystemError)

	tests := []struct {
		name string
		auth *authStub
	}{
		{"the service returned an error", &authStub{userID: "", err: serviceError}},
		{"the service returned no response and no error", &authStub{userID: "user-1", response: nil}},
	}

	build := func(a services.AuthService, r services.RedisService) func(ziface.IRequest) {
		return (&AuthTokenRouter{AuthService: a, RedisService: r}).Handle
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := newFakeConnection(7)
			redis := &fakeRedisService{}

			build(test.auth, redis)(&fakeRequest{conn: conn, data: []byte(`{"id":"x"}`), msgID: constants.RpcAuthByToken})

			want := serviceError
			if test.auth.err == nil {
				want = utils.NewApiError(utils.SystemError)
			}
			assertSingleMessage(t, conn, constants.RpcError, want)
			if len(redis.sessions) != 0 {
				t.Errorf("a failed sign-in stored a session: %+v", redis.sessions)
			}
			if value, ok := conn.props["userId"]; ok {
				t.Errorf("a failed sign-in set userId to %v", value)
			}
		})
	}
}

func TestAuthRouters_GiveUpWhenTheSocketIsGone(t *testing.T) {
	t.Run("the answer cannot be written", func(t *testing.T) {
		conn := newFakeConnection(7)
		msgID := uint32(constants.RpcAuthByToken)
		conn.failFor = &msgID
		redis := &fakeRedisService{}

		(&AuthTokenRouter{AuthService: &authStub{userID: "user-1", response: []byte(`{}`)}, RedisService: redis}).
			Handle(&fakeRequest{conn: conn, data: []byte(`{"id":"x"}`), msgID: msgID})

		if len(conn.props) != 0 || len(redis.sessions) != 0 {
			t.Errorf("a connection that could not be written to was still registered: props=%v sessions=%+v",
				conn.props, redis.sessions)
		}
	})

	t.Run("the error cannot be written", func(t *testing.T) {
		conn := newFakeConnection(7)
		errorID := uint32(constants.RpcError)
		conn.failFor = &errorID

		(&AuthTokenRouter{AuthService: &authStub{err: utils.NewApiError(utils.SystemError)}, RedisService: &fakeRedisService{}}).
			Handle(&fakeRequest{conn: conn, data: []byte(`{"id":"x"}`), msgID: constants.RpcAuthByToken})

		if len(conn.sent) != 0 {
			t.Errorf("the router kept writing after a failed write: %+v", conn.sent)
		}
	})
}

func TestPingRouter_AnswersPongOnItsOwnMessageID(t *testing.T) {
	conn := newFakeConnection(3)

	(&PingRouter{}).Handle(&fakeRequest{conn: conn, data: []byte("ping"), msgID: constants.RpcPing})

	assertSingleMessage(t, conn, 1000, []byte("pong"))
}

// A connection with no userId yet is a client that asked for its account before signing in, and it
// gets an error that says so rather than a lookup for the empty string.
func TestGetUserRouter_NeedsTheUserIdTheConnectionShouldAlreadyHave(t *testing.T) {
	conn := newFakeConnection(4)
	users := &userStub{}

	(&GetUserRouter{UserService: users}).Handle(&fakeRequest{conn: conn, data: nil, msgID: constants.RpcGetUserAccount})

	assertSingleMessage(t, conn, constants.RpcError, utils.NewApiError(utils.UserIdContextError))
	if len(users.asked) != 0 {
		t.Errorf("the service was asked about %v", users.asked)
	}
}

func TestGetUserRouter_AnswersWithTheAccount(t *testing.T) {
	conn := newFakeConnection(4)
	conn.props["userId"] = "user-1"
	users := &userStub{response: []byte(`{"userId":"user-1"}`)}

	(&GetUserRouter{UserService: users}).Handle(&fakeRequest{conn: conn, data: nil, msgID: constants.RpcGetUserAccount})

	assertSingleMessage(t, conn, constants.RpcGetUserAccount, []byte(`{"userId":"user-1"}`))
	if len(users.asked) != 1 || users.asked[0] != "user-1" {
		t.Errorf("the service was asked about %v, want [user-1]", users.asked)
	}
}

func TestGetUserRouter_ReportsAServiceFailureOnce(t *testing.T) {
	conn := newFakeConnection(4)
	conn.props["userId"] = "user-1"
	users := &userStub{err: utils.NewApiError(utils.ItemNotFoundError)}

	(&GetUserRouter{UserService: users}).Handle(&fakeRequest{conn: conn, data: nil, msgID: constants.RpcGetUserAccount})

	assertSingleMessage(t, conn, constants.RpcError, utils.NewApiError(utils.ItemNotFoundError))
}

type userStub struct {
	response []byte
	err      []byte
	asked    []string
}

func (u *userStub) GetUserByUserId(userID string) ([]byte, []byte) {
	u.asked = append(u.asked, userID)
	return u.response, u.err
}
