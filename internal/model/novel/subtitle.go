package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SubtitleType 字幕类型
type SubtitleType string

const (
	SubtitleTypeShot  SubtitleType = "shot"  // 分镜字幕（关联镜头ID）
	SubtitleTypeNovel SubtitleType = "novel" // 完整字幕（关联剧本ID）
)

// SubtitleFormat 字幕格式
type SubtitleFormat string

const (
	SubtitleFormatASS SubtitleFormat = "ass" // ASS 格式
	SubtitleFormatSRT SubtitleFormat = "srt" // SRT 格式
	SubtitleFormatVTT SubtitleFormat = "vtt" // VTT 格式
)

// String 返回格式的字符串表示
func (f SubtitleFormat) String() string {
	return string(f)
}

// Subtitle 字幕实体
// 说明：支持两种类型：分镜字幕（关联镜头ID）和完整字幕（关联剧本ID）
type Subtitle struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 字幕ID（UUID）

	// 关联字段
	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID（两种类型都需要）
	UserID  string `bson:"user_id" json:"user_id"`   // 用户ID

	// 类型和关联（根据 SubtitleType 确定哪个字段不为空）
	SubtitleType SubtitleType `bson:"subtitle_type" json:"subtitle_type"`               // 字幕类型：shot（分镜字幕）、novel（完整字幕）
	ShotID       string       `bson:"shot_id,omitempty" json:"shot_id,omitempty"`       // 关联的镜头ID（分镜字幕时必填）
	ChapterID    string       `bson:"chapter_id,omitempty" json:"chapter_id,omitempty"` // 关联的章节ID（分镜字幕时可选，用于查询）

	// 内容信息
	SubtitleResourceID string         `bson:"subtitle_resource_id" json:"subtitle_resource_id"` // 字幕文件的 resource_id
	Format             SubtitleFormat `bson:"format" json:"format"`                             // 字幕格式：ass, srt, vtt
	Prompt             string         `bson:"prompt,omitempty" json:"prompt,omitempty"`         // 生成字幕时使用的提示词/参数（字幕生成参数配置）

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

// Collection 返回集合名称
func (s *Subtitle) Collection() string {
	return "subtitles"
}

// EnsureIndexes 创建和维护索引
func (s *Subtitle) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(s.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "subtitle_type", Value: 1}},
			Options: options.Index().SetName("idx_subtitle_type"),
		},
		{
			// 分镜字幕索引（支持多版本）
			Keys:    bson.D{{Key: "shot_id", Value: 1}, {Key: "subtitle_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"shot_id": bson.M{"$exists": true, "$type": "string"}, "subtitle_type": SubtitleTypeShot}).SetName("idx_shot_type_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "shot_id", Value: 1}},
			Options: options.Index().SetName("idx_shot_id"),
		},
		{
			// 完整字幕索引（支持多版本）
			Keys:    bson.D{{Key: "novel_id", Value: 1}, {Key: "subtitle_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"subtitle_type": SubtitleTypeNovel}).SetName("idx_novel_type_version_unique"),
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
