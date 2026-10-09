package handlers

import (
	"log"
	"sync"
	"zinx-server/internal/services"

	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/zlog"
	"github.com/aceld/zinx/znet"
)

// ChatRouter joins a connection to the chat room it asked for. Every connection is handled on its
// own goroutine, so the room map is shared state and is guarded here.
type ChatRouter struct {
	znet.BaseRouter

	mu    sync.Mutex
	rooms map[string]*services.ChatRoom
}

// room returns the named chat room, creating it - and starting its broadcast loop - on first use.
//
// The map is created on demand rather than in the literal the router is registered with, because
// a nil map cannot be written to: the first client to join a chat used to panic the process with
// "assignment to entry in nil map". The mutex is what keeps two clients joining at the same time
// from being a concurrent map write, which Go turns into an unrecoverable crash.
func (cr *ChatRouter) room(name string) *services.ChatRoom {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.rooms == nil {
		cr.rooms = make(map[string]*services.ChatRoom)
	}
	existing, exists := cr.rooms[name]
	if !exists {
		existing = services.NewChatRoom(name)
		cr.rooms[name] = existing
		go existing.HandleMessages()
	}
	return existing
}

// Handle reads one message from the connection and broadcasts it to the room.
func (cr *ChatRouter) Handle(request ziface.IRequest) {
	zlog.Info("ChatHandler Handle")

	wsConn := request.GetConnection().GetWsConn()
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		log.Println("Read message error:", err)
		return
	}

	room := cr.room("default")
	room.AddClient(wsConn)
	defer room.RemoveClient(wsConn)

	room.Broadcast <- msg
}
