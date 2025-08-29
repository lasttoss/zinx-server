package routers

import (
	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/zlog"
	"github.com/aceld/zinx/znet"
	"github.com/robfig/cron/v3"
	"zinx-server/internal/configs"
	"zinx-server/internal/constants"
	"zinx-server/internal/filters"
	"zinx-server/internal/handlers"
	"zinx-server/internal/repositories"
	"zinx-server/internal/services"
)

func Route() {
	zlog.Debug("route start")
	baseRouter := znet.BaseRouter{}

	s := znet.NewServer()

	redisService := services.NewRedisService(configs.RedisClient)

	userRepository := repositories.NewUserRepository(configs.MongoDB.UserCollection)
	authService := services.NewAuthService(userRepository, configs.ServerConfig.Google.ClientId,
		configs.ServerConfig.Apple.GoogleClientId, configs.ServerConfig.Apple.ClientId)

	userService := services.NewUserService(userRepository)

	s.SetOnConnStart(OnConnectionAdd)
	s.SetOnConnStop(OnConnectionLost)

	// unauthorized msgId < 1100
	unAuthorizationRpcMaps := map[uint32]ziface.IRouter{
		constants.RpcPing: &handlers.PingRouter{},
		constants.RpcAuthByToken: &handlers.AuthTokenRouter{
			BaseRouter:   znet.BaseRouter{},
			AuthService:  authService,
			RedisService: redisService,
		},
		constants.RpcAuthByDevice: &handlers.AuthDeviceRouter{
			BaseRouter:   baseRouter,
			AuthService:  authService,
			RedisService: redisService},
		constants.RpcAuthByGoogle: &handlers.AuthGoogleRouter{
			BaseRouter:   baseRouter,
			AuthService:  authService,
			RedisService: redisService},
		constants.RpcAuthByApple: &handlers.AuthAppleRouter{
			BaseRouter:   baseRouter,
			AuthService:  authService,
			RedisService: redisService},
		constants.RpcJoinChatRoom: &handlers.ChatRouter{},
	}

	// authorized msgId >= 1100
	authorizationRpcMaps := map[uint32]ziface.IRouter{
		constants.RpcGetUserAccount: &handlers.GetUserRouter{
			BaseRouter:  baseRouter,
			UserService: userService,
		},
	}

	for rpcId, rpcRouter := range unAuthorizationRpcMaps {
		s.AddRouter(rpcId, rpcRouter)
	}

	for rpcId, rpcRouter := range authorizationRpcMaps {
		s.AddRouter(rpcId, rpcRouter)
	}

	s.AddInterceptor(&filters.MyInterceptor{RedisService: redisService})

	redisService.ClearAllSessions()

	job := cron.New(cron.WithSeconds())
	_, _ = job.AddFunc("*/5 * * * * *", func() {
		zlog.Info("CCU: ", len(s.GetConnMgr().GetAllConnID()))
	})

	job.Start()

	s.Serve()
	zlog.Debug("route serve")
}

func OnConnectionAdd(conn ziface.IConnection) {
	zlog.Debug("OnConnectionAdd ==> ", conn.GetConnID())
}

func OnConnectionLost(conn ziface.IConnection) {
	zlog.Debug("OnConnectionLost ==> ", conn.GetConnID())
}
