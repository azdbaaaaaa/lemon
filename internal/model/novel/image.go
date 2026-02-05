package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Image 图片实体
// 说明：每个 Shot（镜头）对应一张场景图片，图片包含人物和场景的完整画面
type Image struct {
	ID string `bson:"id" json:"id"` // 图片ID（UUID）

	ChapterID   string `bson:"chapter_id" json:"chapter_id"`     // 关联的章节ID
	NarrationID string `bson:"narration_id" json:"narration_id"` // 关联的解说ID
	WorkflowID  string `bson:"workflow_id" json:"workflow_id"`   // 关联的工作流ID

	SceneNumber string `bson:"scene_number" json:"scene_number"` // 场景编号（字符串，如 "1"）
	ShotNumber  string `bson:"shot_number" json:"shot_number"`   // 镜头编号（字符串，如 "1"）

	ImageResourceID string `bson:"image_resource_id" json:"image_resource_id"` // 图片文件的 resource_id
	CharacterName   string `bson:"character_name" json:"character_name"`       // 角色名称（镜头中的主要角色）

	Prompt string `bson:"prompt,omitempty" json:"prompt,omitempty"` // 生成图片时使用的完整 prompt

	Version  int    `bson:"version" json:"version"`   // 版本号（用于支持多版本，默认 1）
	Status   TaskStatus `bson:"status" json:"status"`     // 状态：pending, completed, failed
	Sequence int    `bson:"sequence" json:"sequence"` // 序号（用于排序，按场景和镜头编号排序）

	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (i *Image) Collection() string { return "images" }

// EnsureIndexes 创建和维护索引
func (i *Image) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(i.Collection())
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
			Keys:    bson.D{{Key: "narration_id", Value: 1}},
			Options: options.Index().SetName("idx_narration_id"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "scene_number", Value: 1}, {Key: "shot_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_chapter_scene_shot_unique"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "sequence", Value: 1}},
			Options: options.Index().SetName("idx_sequence"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
