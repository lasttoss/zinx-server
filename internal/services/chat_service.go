package services

import (
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type ChatRoom struct {
	Name      string
	Clients   map[*websocket.Conn]bool
	Broadcast chan []byte
	Mutex     sync.Mutex
}

// NewChatRoom returns an empty room with its broadcast channel ready.
func NewChatRoom(name string) *ChatRoom {
	return &ChatRoom{
		Name:      name,
		Clients:   make(map[*websocket.Conn]bool),
		Broadcast: make(chan []byte),
	}
}

// AddClient puts a connection in the room. Safe to call from any goroutine.
func (cr *ChatRoom) AddClient(conn *websocket.Conn) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	cr.Clients[conn] = true
}

// RemoveClient takes a connection out of the room, if it is still in it.
func (cr *ChatRoom) RemoveClient(conn *websocket.Conn) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	delete(cr.Clients, conn)
}

// BroadcastMessage writes to every client. A client that cannot be written to is closed and
// dropped here, which is the only place a dead connection is noticed.
func (cr *ChatRoom) BroadcastMessage(message []byte) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	for client := range cr.Clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error broadcasting message: %v", err)
			client.Close()
			delete(cr.Clients, client)
		}
	}
}

// HandleMessages drains the Broadcast channel until it is closed. Run it as a goroutine per room.
func (cr *ChatRoom) HandleMessages() {
	for message := range cr.Broadcast {
		cr.BroadcastMessage(message)
	}
}
