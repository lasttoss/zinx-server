package configs

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
)

var RedisClient *redis.Client

func ConnectRedis() {
	opts, err := redis.ParseURL(ServerConfig.Redis.Url)
	if err != nil {
		log.Fatalf("redis connect err: %v", err)
	}
	RedisClient = redis.NewClient(opts)

	ctx := context.Background()
	err = RedisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatalf("redis connect err: %v", err)
	}
	log.Println("redis connect success")
}
