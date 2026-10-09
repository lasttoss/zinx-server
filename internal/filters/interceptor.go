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
		// The connection must be authenticated before it can use the authorized
		// message range. Without this guard the type assertion below would panic on
		// a nil property and take the whole server process down.
		property, propertyErr := conn.GetProperty("userId")
		if propertyErr != nil {
			i.reject(conn, iRequest.GetMsgID(), utils.InvalidContextError)
			return chain.Proceed(chain.Request())
		}
		userId, ok := property.(string)
		if !ok || userId == "" {
			i.reject(conn, iRequest.GetMsgID(), utils.InvalidContextError)
			return chain.Proceed(chain.Request())
		}

		sessionId, ok := i.RedisService.GetSession(userId)
		if !ok {
			i.reject(conn, iRequest.GetMsgID(), utils.SystemError)
			return chain.Proceed(chain.Request())
		}

		if conn.GetConnID() != sessionId {
			i.reject(conn, iRequest.GetMsgID(), utils.AnotherDeviceLoginError)
			return chain.Proceed(chain.Request())
		}
	}
	return chain.Proceed(chain.Request())
}

// reject sends an api error and closes the connection.
func (i *MyInterceptor) reject(conn ziface.IConnection, msgId uint32, code utils.ErrorCode) {
	if err := conn.SendMsg(constants.RpcError, utils.NewApiError(code)); err != nil {
		utils.NewSystemError(msgId)
	}
	conn.Stop()
}
