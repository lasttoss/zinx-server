package filters

import (
	"github.com/aceld/zinx/ziface"
	"zinx-server/internal/constants"
	"zinx-server/internal/services"
	"zinx-server/internal/utils"
)

type MyInterceptor struct {
	services.RedisService
}

func (i *MyInterceptor) Intercept(chain ziface.IChain) ziface.IcResp {
	request := chain.Request()
	iRequest := request.(ziface.IRequest)

	// check authorization login multi device
	if iRequest.GetMsgID() >= 1100 {
		conn := iRequest.GetConnection()
		userId, err := conn.GetProperty("userId")
		if err != nil {
			err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidContextError))
			if err != nil {
				utils.NewSystemError(iRequest.GetMsgID())
			}
			conn.Stop()
		}

		sessionId, ok := i.RedisService.GetSession(userId.(string))
		if !ok {
			err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.SystemError))
			if err != nil {
				utils.NewSystemError(iRequest.GetMsgID())
			}
			conn.Stop()
		}

		if conn.GetConnID() != sessionId {
			err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.AnotherDeviceLoginError))
			if err != nil {
				utils.NewSystemError(iRequest.GetMsgID())
			}
			conn.Stop()
		}
	}
	return chain.Proceed(chain.Request())
}
