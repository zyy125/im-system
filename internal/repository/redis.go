package repository

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
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
		return nil, err
	}
	
	log.Println("Redis client initialized")

	return rdb, nil
}


