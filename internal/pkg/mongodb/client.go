package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"lemon/internal/config"
)

// Client MongoDB 客户端封装
type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

// New 创建 MongoDB 客户端
func New(cfg *config.MongoConfig) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 配置客户端选项
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize)

	// 连接 MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	// 验证连接
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &Client{
		client:   client,
		database: client.Database(cfg.Database),
	}, nil
}

// Database 获取数据库
func (c *Client) Database() *mongo.Database {
	return c.database
}

// Collection 获取集合
func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// Close 关闭连接
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// Client 获取原始客户端
func (c *Client) Client() *mongo.Client {
	return c.client
}
