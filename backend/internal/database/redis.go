package database

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

func NewRedis(addr, password string, db int) (*redis.Client, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := cli.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return cli, nil
}
