package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Shot 镜头实体
// 说明：镜头从 ChapterNarration 中分离出来，独立存储
// 通过 scene_id 关联到 Scene，通过 narration_id 关联到 ChapterNarration
type Shot struct {
	ID          string     `bson:"id" json:"id"`                     // 镜头ID（UUID）
	SceneID     string     `bson:"scene_id" json:"scene_id"`        // 关联的场景ID
	SceneNumber string     `bson:"scene_number" json:"scene_number"` // 场景编号（冗余字段，方便查询，字符串，如 "1"）
	NarrationID string     `bson:"narration_id" json:"narration_id"` // 关联的章节解说ID（冗余字段，方便查询）
	ChapterID   string     `bson:"chapter_id" json:"chapter_id"`    // 关联的章节ID（冗余字段，方便查询）
	UserID      string     `bson:"user_id" json:"user_id"`          // 用户ID（冗余字段，方便查询）
	ShotNumber  string     `bson:"shot_number" json:"shot_number"`  // 镜头编号（字符串，如 "1"，原来的 closeup_number）
	Character   string     `bson:"character,omitempty" json:"character,omitempty"` // 角色名称（特写中的主要角色）
	Narration   string     `bson:"narration" json:"narration"`      // 镜头解说内容
	ScenePrompt string     `bson:"scene_prompt,omitempty" json:"scene_prompt,omitempty"` // 图片prompt描述（包含场景描述+角色描述+行为/事件+构图词，用于生成包含人物和场景的完整画面）
	VideoPrompt string     `bson:"video_prompt,omitempty" json:"video_prompt,omitempty"` // 视频prompt描述（用于生成该镜头的动态视频，描述动态效果，例如"镜头缓慢推进，人物缓缓回头"、"树叶随风飘动，光影斑驳"等）
	Sequence    int        `bson:"sequence" json:"sequence"`        // 序号（在场景中的顺序，从1开始）
	Index       int        `bson:"index" json:"index"`               // 全局索引（在所有镜头中的顺序，从1开始，用于跨场景排序）
	Version     int        `bson:"version" json:"version"`          // 版本号（用于支持多版本，默认 1）
	Status      string     `bson:"status" json:"status"`            // 状态：pending, completed, failed
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (s *Shot) Collection() string {
	return "shots"
}

// EnsureIndexes 创建和维护索引
func (s *Shot) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(s.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "scene_id", Value: 1}},
			Options: options.Index().SetName("idx_scene_id"),
		},
		{
			Keys:    bson.D{{Key: "narration_id", Value: 1}},
			Options: options.Index().SetName("idx_narration_id"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "scene_id", Value: 1}, {Key: "shot_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_scene_shot_unique"),
		},
		{
			Keys:    bson.D{{Key: "sequence", Value: 1}},
			Options: options.Index().SetName("idx_sequence"),
		},
		{
			Keys:    bson.D{{Key: "index", Value: 1}},
			Options: options.Index().SetName("idx_index"),
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
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "scene_number", Value: 1}, {Key: "shot_number", Value: 1}},
			Options: options.Index().SetName("idx_chapter_scene_shot"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "scene_number", Value: 1}, {Key: "shot_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_chapter_scene_shot_unique"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
