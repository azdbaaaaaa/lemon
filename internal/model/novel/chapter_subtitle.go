package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChapterSubtitle 章节字幕实体
// 说明：每个 narration shot 对应一个字幕文件（ASS格式），通过 sequence 与音频片段对应
type ChapterSubtitle struct {
	ID                 string     `bson:"id" json:"id"`                                     // 章节字幕ID（UUID）
	ChapterID          string     `bson:"chapter_id" json:"chapter_id"`                     // 关联的章节ID
	NarrationID        string     `bson:"narration_id" json:"narration_id"`                 // 关联的章节解说ID
	UserID             string     `bson:"user_id" json:"user_id"`                           // 用户ID
	Sequence           int        `bson:"sequence" json:"sequence"`                         // 序号（对应 narration shot 的顺序，从1开始）
	SubtitleResourceID string     `bson:"subtitle_resource_id" json:"subtitle_resource_id"` // 字幕文件的 resource_id
	Format             string     `bson:"format" json:"format"`                             // 字幕格式：ass, srt, vtt
	Prompt             string     `bson:"prompt,omitempty" json:"prompt,omitempty"`         // 生成章节字幕时使用的提示词/参数（字幕生成参数配置）
	Version            int        `bson:"version" json:"version"`                           // 版本号（用于支持多版本，默认 1）
	Status             string     `bson:"status" json:"status"`                             // 状态：pending, completed, failed
	CreatedAt          time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt          *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (c *ChapterSubtitle) Collection() string {
	return "chapter_subtitles"
}

// EnsureIndexes 创建和维护索引
func (c *ChapterSubtitle) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(c.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "narration_id", Value: 1}},
			Options: options.Index().SetName("idx_narration_id"),
		},
		{
			Keys:    bson.D{{Key: "narration_id", Value: 1}, {Key: "sequence", Value: 1}},
			Options: options.Index().SetName("idx_narration_sequence"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
