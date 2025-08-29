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

// NewChatRoom tạo một phòng chat mới
func NewChatRoom(name string) *ChatRoom {
	return &ChatRoom{
		Name:      name,
		Clients:   make(map[*websocket.Conn]bool),
		Broadcast: make(chan []byte),
	}
}

// AddClient thêm một client vào phòng chat
func (cr *ChatRoom) AddClient(conn *websocket.Conn) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	cr.Clients[conn] = true
}

// RemoveClient xóa một client khỏi phòng chat
func (cr *ChatRoom) RemoveClient(conn *websocket.Conn) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	delete(cr.Clients, conn)
}

// BroadcastMessage gửi tin nhắn đến tất cả client trong phòng
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

// HandleMessages xử lý tin nhắn từ channel Broadcast
func (cr *ChatRoom) HandleMessages() {
	for message := range cr.Broadcast {
		cr.BroadcastMessage(message)
	}
}
