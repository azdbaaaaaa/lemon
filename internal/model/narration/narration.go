package narration

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Narration 解说文案实体
// 说明：解说文案单独存储，通过 chapter_id 关联章节
// Content 字段使用 map[string]interface{} 存储结构化数据，而不是 XML 字符串
type Narration struct {
	ID        string                 `bson:"id" json:"id"`                               // 解说文案ID（UUID）
	ChapterID string                 `bson:"chapter_id" json:"chapter_id"`               // 关联的章节ID
	UserID    string                 `bson:"user_id" json:"user_id"`                     // 用户ID
	Content   map[string]interface{} `bson:"content" json:"content"`                     // 解说文案内容（结构化数据）
	Version   string                 `bson:"version,omitempty" json:"version,omitempty"` // 版本号（可选，用于支持多版本）
	Status    string                 `bson:"status" json:"status"`                       // 状态：pending, completed, failed
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time             `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
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
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			// 一个章节只能有一个活跃的解说文案（未删除的）
			Keys: bson.D{
				{Key: "chapter_id", Value: 1},
				{Key: "deleted_at", Value: 1},
			},
			Options: options.Index().SetName("idx_chapter_deleted"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
