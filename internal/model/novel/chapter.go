package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Chapter 章节实体
// 说明：章节以 UUID 为主键，关联 novel_id；解说内容由 Narration/Scene/Shot 等表单独存储。
type Chapter struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 章节ID（UUID）

	// 关联字段
	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID
	UserID  string `bson:"user_id" json:"user_id"`   // 用户ID

	// 基本信息
	Sequence int    `bson:"sequence" json:"sequence"` // 章节序号，从1开始
	Title    string `bson:"title" json:"title"`       // 章节标题

	// 内容信息
	ChapterText string `bson:"chapter_text" json:"chapter_text"` // 章节全文

	// 统计信息
	WordCount int `bson:"word_count" json:"word_count"` // 章节字数

	// 版本管理
	ActiveSceneVersion int `bson:"active_scene_version,omitempty" json:"active_scene_version,omitempty"` // 当前生效的场景版本号（从1开始，0表示未设置）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (c *Chapter) Collection() string { return "chapters" }

// EnsureIndexes 创建和维护索引
func (c *Chapter) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(c.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "novel_id", Value: 1},
				{Key: "sequence", Value: 1},
			},
			Options: options.Index().SetName("uniq_novel_sequence").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
