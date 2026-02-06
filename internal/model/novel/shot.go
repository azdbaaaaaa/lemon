package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Shot 镜头实体
// 说明：镜头独立存储，通过 scene_id 关联到 Scene，通过 chapter_id + version 标识批次
// 不再需要 narration_id，直接通过 chapter_id + version 关联
type Shot struct {
	ID          string     `bson:"id" json:"id"`                     // 镜头ID（UUID）
	SceneID     string `bson:"scene_id" json:"scene_id"`         // 关联的场景ID
	SceneNumber string `bson:"scene_number" json:"scene_number"`  // 场景编号（冗余字段，方便查询，字符串，如 "1"）
	NarrationID string `bson:"narration_id" json:"narration_id"`  // 关联的解说ID（批次标识）
	ChapterID   string `bson:"chapter_id" json:"chapter_id"`     // 关联的章节ID
	NovelID     string `bson:"novel_id" json:"novel_id"`         // 关联的小说ID
	UserID      string `bson:"user_id" json:"user_id"`           // 用户ID（冗余字段，方便查询）
	ShotNumber  string     `bson:"shot_number" json:"shot_number"`  // 镜头编号（字符串，如 "1"，原来的 closeup_number）
	Character   string     `bson:"character,omitempty" json:"character,omitempty"` // 角色名称（特写中的主要角色）
	Image       string     `bson:"image" json:"image"`               // 画面描述
	Narration   string     `bson:"narration" json:"narration"`      // 旁白（镜头解说内容）
	SoundEffect string     `bson:"sound_effect,omitempty" json:"sound_effect,omitempty"` // 音效描述
	Duration    float64    `bson:"duration,omitempty" json:"duration,omitempty"`    // 时长（秒）
	ImagePrompt string     `bson:"image_prompt" json:"image_prompt"` // 镜头图片提示词（用于生成该镜头的图片）
	VideoPrompt string     `bson:"video_prompt" json:"video_prompt"` // 镜头视频提示词（用于生成该镜头的动态视频，描述动态效果，例如"镜头缓慢推进，人物缓缓回头"、"树叶随风飘动，光影斑驳"等）
	CameraMovement string  `bson:"camera_movement,omitempty" json:"camera_movement,omitempty"` // 运镜方式（如：推、拉、摇、移、跟、升降等）
	Sequence    int        `bson:"sequence" json:"sequence"`        // 序号（在场景中的顺序，从1开始）
	Index       int        `bson:"index" json:"index"`               // 全局索引（在所有镜头中的顺序，从1开始，用于跨场景排序）
	Version     int        `bson:"version" json:"version"`          // 版本号（用于支持多版本，默认 1）
	Status      TaskStatus `bson:"status" json:"status"`            // 状态：pending, completed, failed
	ErrorMessage string    `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）
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
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
		{
			Keys:    bson.D{{Key: "scene_id", Value: 1}, {Key: "shot_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_scene_shot_unique"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}, {Key: "scene_number", Value: 1}, {Key: "shot_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_chapter_version_scene_shot_unique"),
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
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
