package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Shot 镜头实体
// 说明：镜头独立存储，通过 scene_id 关联到 Scene，通过 sequence 标识顺序
type Shot struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 镜头ID（UUID）

	// 关联字段
	SceneID   string `bson:"scene_id" json:"scene_id"`     // 关联的场景ID
	ChapterID string `bson:"chapter_id" json:"chapter_id"` // 关联的章节ID
	NovelID   string `bson:"novel_id" json:"novel_id"`     // 关联的小说ID
	UserID    string `bson:"user_id" json:"user_id"`       // 用户ID（冗余字段，方便查询）

	// 内容信息
	Narration        string  `bson:"narration" json:"narration"`                   // 旁白（镜头解说内容）
	Duration         float64 `bson:"duration,omitempty" json:"duration,omitempty"` // 时长（秒）
	FirstImagePrompt string  `bson:"first_image_prompt" json:"first_image_prompt"` // 首图提示词（第一帧图片）
	LastImagePrompt  string  `bson:"last_image_prompt" json:"last_image_prompt"`   // 末图提示词（最后一帧图片）
	VideoPrompt      string  `bson:"video_prompt" json:"video_prompt"`             // 视频提示词（动态视频，描述动态效果）
	Sequence         int     `bson:"sequence" json:"sequence"`                     // 序号（在场景中的顺序，从1开始）
	// Index            int     `bson:"index" json:"index"`                           // 全局索引（在所有镜头中的顺序，从1开始，用于跨场景排序）

	// 版本管理
	Version int `bson:"version" json:"version"` // 镜头版本号（从1开始，与场景版本号一致）

	// 关联图片
	FirstImageID string `bson:"first_image_id,omitempty" json:"first_image_id,omitempty"` // 当前绑定的首图ID（用户选择的版本）
	LastImageID  string `bson:"last_image_id,omitempty" json:"last_image_id,omitempty"`   // 当前绑定的尾图ID（用户选择的版本）

	// 状态信息
	Status       TaskStatus `bson:"status" json:"status"`                                   // 状态：pending, completed, failed
	ErrorMessage string     `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (s *Shot) Collection() string {
	return "shots"
}

// EnsureIndexes 创建和维护索引
func (s *Shot) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(s.Collection())

	// 删除旧的唯一索引（如果存在）
	// 从 (scene_id, sequence) 迁移到 (chapter_id, version, sequence)
	oldIndexName := "idx_scene_sequence_unique"
	_, err := coll.Indexes().DropOne(ctx, oldIndexName)
	if err != nil && err != mongo.ErrNoDocuments {
		// 忽略索引不存在的错误，其他错误记录但不阻止继续
		// 注意：MongoDB 驱动可能返回不同的错误类型，这里只处理常见的
	}

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
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}, {Key: "sequence", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_chapter_version_sequence_unique"),
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
	_, err = coll.Indexes().CreateMany(ctx, indexes)
	return err
}
