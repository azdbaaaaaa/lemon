package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes 创建索引
func EnsureIndexes(db *mongo.Database) error {
	ctx := context.Background()

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

	if _, err := convColl.Indexes().CreateMany(ctx, convIndexes); err != nil {
		return err
	}

	return nil
}
