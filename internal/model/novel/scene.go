package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Scene 场景实体
// 说明：场景独立存储，通过 chapter_id + version 标识批次
// 不再需要 narration_id，直接通过 chapter_id + version 关联
type Scene struct {
	ID              string     `bson:"id" json:"id"`                                                   // 场景ID（UUID）
	NarrationID     string     `bson:"narration_id" json:"narration_id"`                               // 关联的解说ID（批次标识）
	ChapterID       string     `bson:"chapter_id" json:"chapter_id"`                                   // 关联的章节ID
	NovelID         string     `bson:"novel_id" json:"novel_id"`                                       // 关联的小说ID
	UserID          string     `bson:"user_id" json:"user_id"`                                         // 用户ID（冗余字段，方便查询）
	SceneNumber     string     `bson:"scene_number" json:"scene_number"`                               // 场景编号（字符串，如 "1"）
	Description     string     `bson:"description" json:"description"`                                 // 场景详细描述
	ImagePrompt     string     `bson:"image_prompt" json:"image_prompt"`                               // 场景图片提示词
	ImageResourceID string     `bson:"image_resource_id,omitempty" json:"image_resource_id,omitempty"` // 场景图片的 resource_id
	Narration       string     `bson:"narration,omitempty" json:"narration,omitempty"`                 // 场景级别的解说内容（可选）
	Sequence        int        `bson:"sequence" json:"sequence"`                                       // 序号（在解说中的顺序，从1开始）
	Version         int        `bson:"version" json:"version"`                                         // 版本号（用于支持多版本，默认 1）
	Status          TaskStatus `bson:"status" json:"status"`                                           // 状态：pending, completed, failed
	ErrorMessage    string     `bson:"error_message,omitempty" json:"error_message,omitempty"`         // 错误信息（失败时）
	CreatedAt       time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (s *Scene) Collection() string {
	return "scenes"
}

// EnsureIndexes 创建和维护索引
func (s *Scene) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(s.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}, {Key: "scene_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_chapter_version_scene_unique"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
		{
			Keys:    bson.D{{Key: "sequence", Value: 1}},
			Options: options.Index().SetName("idx_sequence"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
