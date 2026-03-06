package repository

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func InitRedisClient(ctx context.Context, addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(
		&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
			PoolSize: 10,
			MinIdleConns: 5,
		},
	)

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis ping failed: %v", err)
	}
	
	log.Println("Redis client initialized")

	return &RedisClient{Client: rdb}
}


