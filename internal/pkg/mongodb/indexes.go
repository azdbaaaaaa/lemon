package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
	"lemon/internal/model/resource"
)

// EnsureIndexes 创建所有模型的索引
// 这是一个统一的入口，用于在应用启动时创建所有模型的索引
// 如果模型实现了 Model 接口，会自动调用其 EnsureIndexes 方法
// 对于尚未迁移到新接口的模型，仍然在这里手动创建索引
func EnsureIndexes(db *mongo.Database) error {
	ctx := context.Background()

	// 使用 Model 接口的模型
	models := []Model{
		&resource.Resource{},
		&resource.UploadSession{},
		&novel.Novel{},
		&novel.Chapter{},
		&novel.Narration{},
		&novel.Audio{},
		&novel.Subtitle{},
		&novel.Character{},
		&novel.SceneShotImage{},
	}

	// 为实现了 Model 接口的模型创建索引
	if err := EnsureAllIndexes(ctx, db, models...); err != nil {
		return err
	}

	// 尚未迁移到新接口的模型，手动创建索引
	// TODO: 后续可以将这些模型也迁移到 Model 接口

	// conversations 集合索引
	convColl := db.Collection("conversations")
	convIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}, bson.E{Key: "updated_at", Value: -1}},
			Options: options.Index().SetName("idx_user_updated"),
		},
		{
			Keys:    bson.D{bson.E{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created"),
		},
	}

	if err := CreateIndexes(ctx, convColl, convIndexes); err != nil {
		return err
	}

	// users 集合索引
	// 参考: docs/design/auth/AUTH_DESIGN.md - 10. 数据库索引
	userColl := db.Collection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "username", Value: 1}},
			Options: options.Index().SetName("idx_username").SetUnique(true),
		},
		{
			Keys:    bson.D{bson.E{Key: "email", Value: 1}},
			Options: options.Index().SetName("idx_email").SetUnique(true),
		},
		{
			Keys:    bson.D{bson.E{Key: "role", Value: 1}, bson.E{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_role_status"),
		},
		{
			Keys:    bson.D{bson.E{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
	}

	if err := CreateIndexes(ctx, userColl, userIndexes); err != nil {
		return err
	}

	// refresh_tokens 集合索引
	// 参考: docs/design/auth/AUTH_DESIGN.md - 10. 数据库索引
	refreshTokenColl := db.Collection("refresh_tokens")
	refreshTokenIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
		{
			Keys:    bson.D{bson.E{Key: "token", Value: 1}},
			Options: options.Index().SetName("idx_token").SetUnique(true),
		},
		{
			Keys:    bson.D{bson.E{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("idx_expires_at").SetExpireAfterSeconds(0), // TTL索引，自动删除过期token
		},
	}

	if err := CreateIndexes(ctx, refreshTokenColl, refreshTokenIndexes); err != nil {
		return err
	}

	return nil
}
