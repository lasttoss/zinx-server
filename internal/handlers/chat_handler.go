package handlers

import (
	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/zlog"
	"github.com/aceld/zinx/znet"
	"log"
	"zinx-server/internal/services"
)

type ChatRouter struct {
	znet.BaseRouter
	Rooms map[string]*services.ChatRoom
}

// Handle xử lý tin nhắn từ client
func (cr *ChatRouter) Handle(request ziface.IRequest) {
	zlog.Info("ChatHandler Handle")
	wsConn := request.GetConnection().GetWsConn()
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		log.Println("Read message error:", err)
		return
	}

	roomName := "default"

	room, exists := cr.Rooms[roomName]
	if !exists {
		room = services.NewChatRoom(roomName)
		cr.Rooms[roomName] = room
		go room.HandleMessages()
	}

	room.AddClient(wsConn)
	defer room.RemoveClient(wsConn)

	room.Broadcast <- msg
}
