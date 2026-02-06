package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NarrationType 旁白类型
type NarrationType string

const (
	NarrationTypeNarration NarrationType = "narration" // 旁白（解说）类型
	NarrationTypeDialogue  NarrationType = "dialogue"  // 真人对话类型
)

// NovelStyle 剧本风格
type NovelStyle string

const (
	NovelStyleAnime NovelStyle = "anime" // 漫剧（动画风格）
	NovelStyleLive  NovelStyle = "live"  // 真人剧（真人风格）
	NovelStyleMixed NovelStyle = "mixed" // 混合风格
)

// Novel 小说实体（主表）
// 用途：关联上传资源（resource_id），作为整个创作流程的核心实体
type Novel struct {
	ID string `bson:"id" json:"id"` // 小说ID（UUID）

	UserID string `bson:"user_id" json:"user_id"`

	// 关联上传的原始资源
	ResourceID string `bson:"resource_id" json:"resource_id"`

	// 小说元数据
	Title       string `bson:"title,omitempty" json:"title,omitempty"`             // 小说名称
	Author      string `bson:"author,omitempty" json:"author,omitempty"`           // 作者
	Description string `bson:"description,omitempty" json:"description,omitempty"` // 简介

	// 创作配置
	NarrationType NarrationType `bson:"narration_type" json:"narration_type"` // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
	Style         NovelStyle    `bson:"style" json:"style"`                   // 风格：anime（漫剧）、live（真人剧）、mixed（混合）

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
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "resource_id", Value: 1}},
			Options: options.Index().SetName("idx_resource_id"),
		},
		{
			Keys:    bson.D{{Key: "style", Value: 1}},
			Options: options.Index().SetName("idx_style"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
