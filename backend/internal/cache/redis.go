package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"promo-engine/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func Init(cfg *config.Config) {
	Client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
	})
}

func Get(ctx context.Context, key string, value interface{}) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func Set(ctx context.Context, key string, value interface{}, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, time.Duration(ttl)*time.Second).Err()
}

func Delete(ctx context.Context, key string) error {
	return Client.Del(ctx, key).Err()
}

func Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return Client.Publish(ctx, channel, data).Err()
}

func Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return Client.Subscribe(ctx, channel)
}
