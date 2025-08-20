package services

import (
	"context"
	"fmt"
	"github.com/aceld/zinx/zlog"
	"github.com/go-redis/redis/v8"
	"zinx-server/internal/constants"
)

type RedisService interface {
	ClearAllSessions()
	SaveSession(userId string, sessionId uint64)
	GetSession(userId string) (uint64, bool)
}

type redisService struct {
	redisClient *redis.Client
}

func (r redisService) ClearAllSessions() {
	zlog.Debug("[REDIS] clear all sessions")
	ctx := context.Background()
	pattern := fmt.Sprintf("%s*", constants.SessionCached)
	keys, err := r.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}
	for _, key := range keys {
		zlog.Info("Clear session from: ", key)
		r.redisClient.Del(ctx, key)
	}
}

func (r redisService) SaveSession(userId string, sessionId uint64) {
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", constants.SessionCached, userId)
	r.redisClient.Set(ctx, key, sessionId, 0)
}

func (r redisService) GetSession(userId string) (uint64, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", constants.SessionCached, userId)
	val, err := r.redisClient.Get(ctx, key).Int64()
	if err != nil {
		return 0, false
	}
	return uint64(val), true
}

func NewRedisService(redisClient *redis.Client) RedisService {
	return &redisService{
		redisClient: redisClient,
	}
}
