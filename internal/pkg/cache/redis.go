package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"lemon/internal/config"
)

// RedisCache Redis 缓存封装
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建 Redis 缓存客户端
func NewRedisCache(cfg *config.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{client: client}, nil
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string, dest any) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists 检查 key 是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

// Close 关闭连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Client 获取原始客户端
func (c *RedisCache) Client() *redis.Client {
	return c.client
}

// 常用 key 模式
const (
	ConversationCacheKeyPrefix = "conv:"
	ConversationCacheTTL       = 30 * time.Minute
)

// ConversationCacheKey 生成对话缓存 key
func ConversationCacheKey(id string) string {
	return ConversationCacheKeyPrefix + id
}
