package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Model MongoDB 模型接口
// 所有需要管理索引的模型都应该实现这个接口
type Model interface {
	// Collection 返回集合名称
	Collection() string

	// EnsureIndexes 创建和维护索引
	// db: MongoDB 数据库实例
	// 返回: 错误信息
	EnsureIndexes(ctx context.Context, db *mongo.Database) error
}

// EnsureAllIndexes 为所有模型创建索引
// 这是一个统一的入口，用于在应用启动时创建所有模型的索引
func EnsureAllIndexes(ctx context.Context, db *mongo.Database, models ...Model) error {
	for _, model := range models {
		if err := model.EnsureIndexes(ctx, db); err != nil {
			return err
		}
	}
	return nil
}

// CreateIndexes 辅助函数：创建索引
// 用于简化模型实现中的索引创建逻辑
func CreateIndexes(ctx context.Context, coll *mongo.Collection, indexes []mongo.IndexModel) error {
	if len(indexes) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}

// CreateIndex 辅助函数：创建单个索引
func CreateIndex(ctx context.Context, coll *mongo.Collection, index mongo.IndexModel) error {
	_, err := coll.Indexes().CreateOne(ctx, index)
	return err
}
