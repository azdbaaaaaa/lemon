package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Narration 解说实体
// 说明：解说单独存储，通过 chapter_id 关联章节
// 注意：Characters 存储在 Character 表中，Scenes 存储在 Scene 表中，Shots 存储在 Shot 表中
// 此表只存储解说的基本信息和元数据
type Narration struct {
	ID         string     `bson:"id" json:"id"`                             // 解说ID（UUID）
	ChapterID  string     `bson:"chapter_id" json:"chapter_id"`             // 关联的章节ID
	WorkflowID string     `bson:"workflow_id" json:"workflow_id"`           // 关联的工作流ID
	UserID     string     `bson:"user_id" json:"user_id"`                   // 用户ID
	Prompt     string     `bson:"prompt,omitempty" json:"prompt,omitempty"` // 生成解说时使用的提示词
	Version    int        `bson:"version" json:"version"`                   // 版本号（用于支持多版本，默认 1）
	Status     TaskStatus `bson:"status" json:"status"`                     // 状态：pending, completed, failed
	CreatedAt  time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (n *Narration) Collection() string {
	return "narrations"
}

// EnsureIndexes 创建和维护索引
func (n *Narration) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(n.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "workflow_id", Value: 1}},
			Options: options.Index().SetName("idx_workflow_id"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			// 一个章节只能有一个活跃的章节解说（未删除的）
			Keys: bson.D{
				{Key: "chapter_id", Value: 1},
				{Key: "deleted_at", Value: 1},
			},
			Options: options.Index().SetName("idx_chapter_deleted"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
