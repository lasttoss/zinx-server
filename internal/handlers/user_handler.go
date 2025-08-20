package handlers

import (
	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/znet"
	"zinx-server/internal/constants"
	"zinx-server/internal/services"
	"zinx-server/internal/utils"
)

type GetUserRouter struct {
	znet.BaseRouter
	services.UserService
}

func (s *GetUserRouter) Handle(request ziface.IRequest) {
	conn := request.GetConnection()
	userId, connErr := conn.GetProperty("userId")
	if connErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.UserIdContextError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}

	response, serviceErr := s.UserService.GetUserByUserId(userId.(string))
	if serviceErr != nil {
		err := conn.SendMsg(constants.RpcError, serviceErr)
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}

	err := conn.SendMsg(request.GetMsgID(), response)
	if err != nil {
		utils.NewSystemError(request.GetMsgID())
		return
	}
	return
}
