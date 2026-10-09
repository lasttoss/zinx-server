package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// connectRoom starts a server that puts every websocket connection it accepts into the room, then
// dials it. It returns both ends: the client end to read what the room broadcasts, and the server
// end because that is the pointer the room keys its clients by.
func connectRoom(t *testing.T, room *ChatRoom) (server, client *websocket.Conn) {
	t.Helper()

	registered := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		room.AddClient(conn)
		select {
		case registered <- conn:
		default:
		}
		// hold the connection open until the client goes away
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial the chat server: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	select {
	case server = <-registered:
	case <-time.After(3 * time.Second):
		t.Fatal("the server never registered the connection")
	}
	return server, client
}

func roomSize(room *ChatRoom) int {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	return len(room.Clients)
}

func read(t *testing.T, client *websocket.Conn) string {
	t.Helper()

	if err := client.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	_, msg, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("read what the room broadcast: %v", err)
	}
	return string(msg)
}

func TestAddAndRemoveClient(t *testing.T) {
	room := NewChatRoom("default")
	if room.Name != "default" {
		t.Errorf("name = %q, want default", room.Name)
	}
	if roomSize(room) != 0 {
		t.Fatalf("a fresh room has %d clients, want 0", roomSize(room))
	}

	server, _ := connectRoom(t, room)
	if roomSize(room) != 1 {
		t.Fatalf("clients = %d after one join, want 1", roomSize(room))
	}

	room.RemoveClient(server)
	if roomSize(room) != 0 {
		t.Fatalf("clients = %d after the client left, want 0", roomSize(room))
	}
	// a connection that leaves twice must not take somebody else out of the room with it
	room.RemoveClient(server)
	if roomSize(room) != 0 {
		t.Errorf("clients = %d after a second remove, want 0", roomSize(room))
	}
}

func TestBroadcastReachesEveryClientInTheRoom(t *testing.T) {
	room := NewChatRoom("default")
	go room.HandleMessages()

	_, first := connectRoom(t, room)
	_, second := connectRoom(t, room)
	if roomSize(room) != 2 {
		t.Fatalf("clients = %d, want 2", roomSize(room))
	}

	room.Broadcast <- []byte("hello room")

	if got := read(t, first); got != "hello room" {
		t.Errorf("first client received %q, want %q", got, "hello room")
	}
	if got := read(t, second); got != "hello room" {
		t.Errorf("second client received %q, want %q", got, "hello room")
	}
}

func TestBroadcastWithNobodyInTheRoomIsANoOp(t *testing.T) {
	room := NewChatRoom("empty")
	room.BroadcastMessage([]byte("nobody is listening"))
	if roomSize(room) != 0 {
		t.Errorf("clients = %d, want 0", roomSize(room))
	}
}

// TestBroadcastDropsAClientThatCannotBeWrittenTo: a phone that walks out of wifi leaves a socket
// that accepts no more writes. The room has to notice and let it go, or every later broadcast
// pays for it.
func TestBroadcastDropsAClientThatCannotBeWrittenTo(t *testing.T) {
	room := NewChatRoom("default")
	go room.HandleMessages()

	_, alive := connectRoom(t, room)
	dead, _ := connectRoom(t, room)
	if roomSize(room) != 2 {
		t.Fatalf("clients = %d, want 2", roomSize(room))
	}

	// Close the socket under the room's side of the dead client. A lost phone looks like this: the
	// connection is gone and every write to it fails. Closing it here rather than from the client
	// end keeps the test deterministic - writing to a socket this process already closed returns an
	// error straight away, instead of depending on how fast the network stack reports a reset.
	if err := dead.UnderlyingConn().Close(); err != nil {
		t.Fatalf("close the dead client: %v", err)
	}

	for i := 0; i < 10 && roomSize(room) > 1; i++ {
		room.BroadcastMessage([]byte("tick"))
		time.Sleep(20 * time.Millisecond)
	}
	if got := roomSize(room); got != 1 {
		t.Fatalf("clients = %d after broadcasting to a dead socket, want 1", got)
	}

	// and the client that is still there keeps being served: skip whatever the probing
	// broadcasts left in its queue, the point is that this message arrives
	room.BroadcastMessage([]byte("still here"))
	got := ""
	for i := 0; i < 10 && got != "still here"; i++ {
		got = read(t, alive)
	}
	if got != "still here" {
		t.Errorf("the surviving client never received the broadcast, it ended on %q", got)
	}
}

func TestAddRemoveUnderConcurrency(t *testing.T) {
	room := NewChatRoom("default")

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); room.AddClient(nil) }()
		go func() { defer wg.Done(); room.RemoveClient(nil) }()
	}
	wg.Wait()
}
