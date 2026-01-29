package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Novel 小说实体（主表）
// 用途：关联上传资源（resource_id），并以 workflow_id 串联整个流程。
type Novel struct {
	ID string `bson:"id" json:"id"` // 小说ID（UUID）

	WorkflowID string `bson:"workflow_id" json:"workflow_id"`
	UserID     string `bson:"user_id" json:"user_id"`

	// 关联上传的原始资源
	ResourceID string `bson:"resource_id" json:"resource_id"`

	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (n *Novel) Collection() string { return "novels" }

// EnsureIndexes 创建和维护索引
func (n *Novel) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(n.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "workflow_id", Value: 1}},
			Options: options.Index().SetName("idx_workflow_id"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "resource_id", Value: 1}},
			Options: options.Index().SetName("idx_resource_id"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
