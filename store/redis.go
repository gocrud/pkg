package store

import "github.com/redis/go-redis/v9"

// Redis 包装 redis.UniversalClient;它不参与关系库事务,不随 ctx 变化。
type Redis struct {
	rdb redis.UniversalClient
}

// NewRedis 的 rdb 不能为空。
func NewRedis(rdb redis.UniversalClient) *Redis {
	if rdb == nil {
		panic("store: NewRedis 的 rdb 不能为空")
	}
	return &Redis{rdb: rdb}
}

func (r *Redis) Client() redis.UniversalClient {
	return r.rdb
}
