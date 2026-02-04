package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Scene 场景实体
// 说明：场景从 ChapterNarration 中分离出来，独立存储
// 通过 narration_id 关联到 ChapterNarration
type Scene struct {
	ID          string    `bson:"id" json:"id"`                     // 场景ID（UUID）
	NarrationID string    `bson:"narration_id" json:"narration_id"` // 关联的章节解说ID
	ChapterID   string    `bson:"chapter_id" json:"chapter_id"`    // 关联的章节ID（冗余字段，方便查询）
	UserID      string    `bson:"user_id" json:"user_id"`          // 用户ID（冗余字段，方便查询）
	SceneNumber string    `bson:"scene_number" json:"scene_number"` // 场景编号（字符串，如 "1"）
	Narration   string    `bson:"narration,omitempty" json:"narration,omitempty"` // 场景级别的解说内容（可选）
	Sequence    int       `bson:"sequence" json:"sequence"`         // 序号（在解说中的顺序，从1开始）
	Version     int       `bson:"version" json:"version"`           // 版本号（用于支持多版本，默认 1）
	Status      string    `bson:"status" json:"status"`            // 状态：pending, completed, failed
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
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
			Keys:    bson.D{{Key: "narration_id", Value: 1}},
			Options: options.Index().SetName("idx_narration_id"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "narration_id", Value: 1}, {Key: "scene_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_narration_scene_unique"),
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
			Keys:    bson.D{{Key: "narration_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_narration_version"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
