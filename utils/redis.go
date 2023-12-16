package utils

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client  *redis.Client
	Context context.Context
}

func NewRedisClient(url string) (RedisClient, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return RedisClient{}, err
	}
	client := redis.NewClient(opts)
	context := context.Background()

	err = client.Ping(context).Err()
	if err != nil {
		return RedisClient{}, err
	}

	result := RedisClient{
		Client:  client,
		Context: context,
	}

	return result, nil
}
