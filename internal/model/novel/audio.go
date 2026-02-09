package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AudioType 音频类型
type AudioType string

const (
	AudioTypeShot  AudioType = "shot"  // 分镜音频（关联镜头ID）
	AudioTypeNovel AudioType = "novel" // 完整音频（关联剧本ID）
)

// Audio 音频实体
// 说明：支持两种类型：分镜音频（关联镜头ID）和完整音频（关联剧本ID）
type Audio struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 音频ID（UUID）

	// 关联字段
	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID（两种类型都需要）
	UserID  string `bson:"user_id" json:"user_id"`   // 用户ID

	// 类型和关联（根据 AudioType 确定哪个字段不为空）
	AudioType AudioType `bson:"audio_type" json:"audio_type"`                     // 音频类型：shot（分镜音频）、novel（完整音频）
	ShotID    string    `bson:"shot_id,omitempty" json:"shot_id,omitempty"`       // 关联的镜头ID（分镜音频时必填）
	ChapterID string    `bson:"chapter_id,omitempty" json:"chapter_id,omitempty"` // 关联的章节ID（分镜音频时可选，用于查询）

	// 内容信息
	AudioResourceID string     `bson:"audio_resource_id" json:"audio_resource_id"` // 音频文件的 resource_id
	Duration        float64    `bson:"duration" json:"duration"`                   // 音频时长（秒）
	Text            string     `bson:"text" json:"text"`                           // 对应的文本内容
	Timestamps      []CharTime `bson:"timestamps" json:"timestamps"`               // 字符级别的时间戳
	Prompt          string     `bson:"prompt,omitempty" json:"prompt,omitempty"`   // 生成音频时使用的提示词/参数（TTS参数配置）

	// 版本信息
	Version int `bson:"version" json:"version"` // 版本号（用于支持多版本，默认 1）

	// 状态信息
	Status       TaskStatus `bson:"status" json:"status"`                                   // 状态：pending, completed, failed
	ErrorMessage string     `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// CharTime 字符时间戳
type CharTime struct {
	Character string  `bson:"character" json:"character"`   // 字符
	StartTime float64 `bson:"start_time" json:"start_time"` // 开始时间（秒）
	EndTime   float64 `bson:"end_time" json:"end_time"`     // 结束时间（秒）
}

// Collection 返回集合名称
func (a *Audio) Collection() string {
	return "audios"
}

// EnsureIndexes 创建和维护索引
func (a *Audio) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(a.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "audio_type", Value: 1}},
			Options: options.Index().SetName("idx_audio_type"),
		},
		{
			// 分镜音频索引（支持多版本）
			Keys:    bson.D{{Key: "shot_id", Value: 1}, {Key: "audio_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"shot_id": bson.M{"$exists": true, "$type": "string"}, "audio_type": AudioTypeShot}).SetName("idx_shot_type_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "shot_id", Value: 1}},
			Options: options.Index().SetName("idx_shot_id"),
		},
		{
			// 完整音频索引（支持多版本）
			Keys:    bson.D{{Key: "novel_id", Value: 1}, {Key: "audio_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"audio_type": AudioTypeNovel}).SetName("idx_novel_type_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_version"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
