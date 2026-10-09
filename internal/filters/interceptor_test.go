package filters

import (
	"errors"
	"testing"

	"github.com/aceld/zinx/ziface"

	"zinx-server/internal/constants"
	"zinx-server/internal/utils"
)

// The interceptor is what stands between a socket and every message in the authorized range, so
// these tests check each way a connection can fail to prove who it is, and that the answer is an
// error message followed by a closed connection.
//
// One piece of behaviour is recorded rather than assumed: after rejecting, the interceptor still
// calls Proceed, so the router runs as well. That is what the code does today; the tests pin it down
// so that changing it is a decision rather than an accident.

type sentMessage struct {
	msgID uint32
	data  []byte
}

type fakeConnection struct {
	ziface.IConnection

	connID  uint64
	props   map[string]any
	sent    []sentMessage
	stopped bool
}

func newFakeConnection(connID uint64, props map[string]any) *fakeConnection {
	if props == nil {
		props = map[string]any{}
	}
	return &fakeConnection{connID: connID, props: props}
}

func (c *fakeConnection) SendMsg(msgID uint32, data []byte) error {
	c.sent = append(c.sent, sentMessage{msgID: msgID, data: data})
	return nil
}

func (c *fakeConnection) GetProperty(key string) (any, error) {
	value, ok := c.props[key]
	if !ok {
		return nil, errors.New("no such property")
	}
	return value, nil
}

func (c *fakeConnection) GetConnID() uint64 { return c.connID }

func (c *fakeConnection) Stop() { c.stopped = true }

type fakeRequest struct {
	ziface.IRequest

	conn  ziface.IConnection
	msgID uint32
}

func (r *fakeRequest) GetConnection() ziface.IConnection { return r.conn }
func (r *fakeRequest) GetMsgID() uint32                  { return r.msgID }

type fakeChain struct {
	ziface.IChain

	req       ziface.IcReq
	proceeded int
}

func (c *fakeChain) Request() ziface.IcReq { return c.req }

func (c *fakeChain) Proceed(ziface.IcReq) ziface.IcResp {
	c.proceeded++
	return nil
}

type fakeRedis struct {
	session uint64
	found   bool
	asked   []string
}

func (f *fakeRedis) ClearAllSessions() {}

func (f *fakeRedis) SaveSession(string, uint64) {}

func (f *fakeRedis) GetSession(userID string) (uint64, bool) {
	f.asked = append(f.asked, userID)
	return f.session, f.found
}

func intercept(t *testing.T, msgID uint32, props map[string]any, redis *fakeRedis) (*fakeConnection, *fakeChain) {
	t.Helper()
	conn := newFakeConnection(7, props)
	chain := &fakeChain{req: &fakeRequest{conn: conn, msgID: msgID}}

	if response := (&MyInterceptor{RedisService: redis}).Intercept(chain); response != nil {
		t.Errorf("Intercept returned %v, want nil", response)
	}
	return conn, chain
}

func assertRejected(t *testing.T, conn *fakeConnection, code utils.ErrorCode) {
	t.Helper()
	if len(conn.sent) != 1 {
		t.Fatalf("the client got %d messages, want exactly 1: %+v", len(conn.sent), conn.sent)
	}
	if conn.sent[0].msgID != constants.RpcError {
		t.Errorf("message id = %d, want %d", conn.sent[0].msgID, constants.RpcError)
	}
	if want := string(utils.NewApiError(code)); string(conn.sent[0].data) != want {
		t.Errorf("body = %s, want %s", conn.sent[0].data, want)
	}
	if !conn.stopped {
		t.Error("the connection was left open after being rejected")
	}
}

// Messages below 1100 are the sign-in range and cannot require a user id: the client does not have
// one yet. They go straight through, and the redis lookup is not even attempted.
func TestIntercept_LetsTheSignInRangeThrough(t *testing.T) {
	for _, msgID := range []uint32{constants.RpcAuthByToken, constants.RpcAuthByDevice, constants.RpcPing} {
		redis := &fakeRedis{}
		conn, chain := intercept(t, msgID, nil, redis)

		if len(conn.sent) != 0 || conn.stopped {
			t.Errorf("message %d was rejected: %+v", msgID, conn.sent)
		}
		if chain.proceeded != 1 {
			t.Errorf("message %d: the chain was proceeded %d times, want 1", msgID, chain.proceeded)
		}
		if len(redis.asked) != 0 {
			t.Errorf("message %d was checked against a session: %v", msgID, redis.asked)
		}
	}
}

// 1100 is the first guarded message id, so the boundary itself is worth a test: an off-by-one here
// either locks out the account screen or opens everything above it.
func TestIntercept_GuardsFromElevenHundredUpwards(t *testing.T) {
	redis := &fakeRedis{}
	conn, chain := intercept(t, 1099, nil, redis)

	if len(conn.sent) != 0 {
		t.Fatalf("1099 was guarded: %+v", conn.sent)
	}
	if chain.proceeded != 1 {
		t.Errorf("1099: the chain was proceeded %d times, want 1", chain.proceeded)
	}

	redis = &fakeRedis{}
	conn, _ = intercept(t, constants.RpcGetUserAccount, nil, redis)
	if len(conn.sent) != 1 {
		t.Fatalf("1100 was not guarded: %+v", conn.sent)
	}
}

func TestIntercept_RejectsAGuardedMessageWithoutAUsableUserId(t *testing.T) {
	tests := []struct {
		name  string
		props map[string]any
	}{
		{"the connection never signed in", nil},
		{"the property is not a user id", map[string]any{"userId": 42}},
		{"the property is an empty user id", map[string]any{"userId": ""}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			redis := &fakeRedis{}
			conn, chain := intercept(t, constants.RpcGetUserAccount, test.props, redis)

			assertRejected(t, conn, utils.InvalidContextError)
			if len(redis.asked) != 0 {
				t.Errorf("a session was looked up for %v", redis.asked)
			}
			if chain.proceeded != 1 {
				t.Errorf("the chain was proceeded %d times, want 1", chain.proceeded)
			}
		})
	}
}

// A session that has expired is not a system problem, but a connection that cannot be checked at all
// is: the connection is closed either way, and the code tells the client which one happened.
func TestIntercept_RejectsAConnectionWhoseSessionIsGone(t *testing.T) {
	redis := &fakeRedis{found: false}
	conn, _ := intercept(t, constants.RpcGetUserAccount, map[string]any{"userId": "user-1"}, redis)

	assertRejected(t, conn, utils.SystemError)
	if len(redis.asked) != 1 || redis.asked[0] != "user-1" {
		t.Errorf("session lookups = %v, want [user-1]", redis.asked)
	}
}

// One account on two devices: the second device gets the connection id it does not own, and the
// answer is the error a client uses to say "you are signed in somewhere else".
func TestIntercept_RejectsASecondDevice(t *testing.T) {
	redis := &fakeRedis{session: 99, found: true}
	conn, _ := intercept(t, constants.RpcGetUserAccount, map[string]any{"userId": "user-1"}, redis)

	assertRejected(t, conn, utils.AnotherDeviceLoginError)
}

func TestIntercept_AllowsTheDeviceThatOwnsTheSession(t *testing.T) {
	redis := &fakeRedis{session: 7, found: true}
	conn, chain := intercept(t, constants.RpcGetUserAccount, map[string]any{"userId": "user-1"}, redis)

	if len(conn.sent) != 0 || conn.stopped {
		t.Errorf("the session's own connection was rejected: sent=%+v stopped=%v", conn.sent, conn.stopped)
	}
	if chain.proceeded != 1 {
		t.Errorf("the chain was proceeded %d times, want 1", chain.proceeded)
	}
	if len(redis.asked) != 1 || redis.asked[0] != "user-1" {
		t.Errorf("session lookups = %v, want [user-1]", redis.asked)
	}
}
