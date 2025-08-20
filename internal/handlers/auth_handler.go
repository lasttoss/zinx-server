package handlers

import (
	"encoding/json"
	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/znet"
	"github.com/go-playground/validator/v10"
	"zinx-server/internal/constants"
	"zinx-server/internal/mappers"
	"zinx-server/internal/services"
	"zinx-server/internal/utils"
)

type AuthDeviceRouter struct {
	znet.BaseRouter
	services.AuthService
	services.RedisService
}

func (p *AuthDeviceRouter) Handle(request ziface.IRequest) {
	conn := request.GetConnection()
	var req mappers.AuthRequest
	jsonErr := json.Unmarshal(request.GetData(), &req)
	if jsonErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}
	validate := validator.New()
	if validateErr := validate.Struct(req); validateErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}

	userId, response, serviceErr := p.AuthByDevice(req)
	if serviceErr != nil {
		err := conn.SendMsg(constants.RpcError, serviceErr)
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
	}

	if response == nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.SystemError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
	}

	err := conn.SendMsg(request.GetMsgID(), response)
	if err != nil {
		utils.NewSystemError(request.GetMsgID())
		return
	}
	conn.SetProperty("userId", userId)
	p.RedisService.SaveSession(userId, conn.GetConnID())
	return
}

type AuthGoogleRouter struct {
	znet.BaseRouter
	services.AuthService
	services.RedisService
}

func (p *AuthGoogleRouter) Handle(request ziface.IRequest) {
	conn := request.GetConnection()
	var req mappers.AuthRequest
	jsonErr := json.Unmarshal(request.GetData(), &req)
	if jsonErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}
	validate := validator.New()
	if validateErr := validate.Struct(req); validateErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}

	userId, response, serviceErr := p.AuthByGoogle(req)
	if serviceErr != nil {
		err := conn.SendMsg(constants.RpcError, serviceErr)
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
	}

	if response == nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.SystemError))
		if err != nil {
			return
		}
	}

	err := conn.SendMsg(request.GetMsgID(), response)
	if err != nil {
		utils.NewSystemError(request.GetMsgID())
		return
	}
	conn.SetProperty("userId", userId)
	p.RedisService.SaveSession(userId, conn.GetConnID())
	return
}

type AuthFacebookRouter struct {
	znet.BaseRouter
	services.RedisService
}

func (p *AuthFacebookRouter) Handle(request ziface.IRequest) {

}

type AuthAppleRouter struct {
	znet.BaseRouter
	services.AuthService
	services.RedisService
}

func (p *AuthAppleRouter) Handle(request ziface.IRequest) {
	conn := request.GetConnection()
	var req mappers.AuthRequest
	jsonErr := json.Unmarshal(request.GetData(), &req)
	if jsonErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}
	validate := validator.New()
	if validateErr := validate.Struct(req); validateErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}

	userId, response, serviceErr := p.AuthByApple(req)
	if serviceErr != nil {
		err := conn.SendMsg(constants.RpcError, serviceErr)
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
	}

	if response == nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.SystemError))
		if err != nil {
			return
		}
	}

	err := conn.SendMsg(request.GetMsgID(), response)
	if err != nil {
		utils.NewSystemError(request.GetMsgID())
		return
	}
	conn.SetProperty("userId", userId)
	p.RedisService.SaveSession(userId, conn.GetConnID())
	return
}

type AuthTokenRouter struct {
	znet.BaseRouter
	services.AuthService
	services.RedisService
}

func (p *AuthTokenRouter) Handle(request ziface.IRequest) {
	conn := request.GetConnection()
	var req mappers.AuthRequest
	jsonErr := json.Unmarshal(request.GetData(), &req)
	if jsonErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}
	validate := validator.New()
	if validateErr := validate.Struct(req); validateErr != nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.InvalidRequestError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
		return
	}
	userId, response, serviceErr := p.AuthByToken(req)
	if serviceErr != nil {
		err := conn.SendMsg(constants.RpcError, serviceErr)
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
	}

	if response == nil {
		err := conn.SendMsg(constants.RpcError, utils.NewApiError(utils.SystemError))
		if err != nil {
			utils.NewSystemError(request.GetMsgID())
			return
		}
	}

	err := conn.SendMsg(request.GetMsgID(), response)
	if err != nil {
		utils.NewSystemError(request.GetMsgID())
		return
	}
	conn.SetProperty("userId", userId)
	p.RedisService.SaveSession(userId, conn.GetConnID())
	return
}
