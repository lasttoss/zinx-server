package handlers

import (
	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/zlog"
	"github.com/aceld/zinx/znet"
	"zinx-server/internal/utils"
)

type PingRouter struct {
	znet.BaseRouter
}

func (p *PingRouter) Handle(request ziface.IRequest) {
	conn := request.GetConnection()
	zlog.Debug("data: ", string(request.GetData()))
	if sendErr := conn.SendMsg(1000, []byte("pong")); sendErr != nil {
		utils.NewSystemError(request.GetMsgID())
		return
	}
}
