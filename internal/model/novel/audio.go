package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Audio 音频实体
// 说明：每个解说的每个镜头会生成一个音频片段，每个片段对应一个 Audio 记录
type Audio struct {
	ID              string     `bson:"id" json:"id"`                               // 音频ID（UUID）
	NarrationID     string     `bson:"narration_id" json:"narration_id"`           // 关联的解说ID
	ChapterID       string     `bson:"chapter_id" json:"chapter_id"`               // 关联的章节ID
	WorkflowID      string     `bson:"workflow_id" json:"workflow_id"`             // 关联的工作流ID
	UserID          string     `bson:"user_id" json:"user_id"`                     // 用户ID
	Sequence        int        `bson:"sequence" json:"sequence"`                   // 音频片段序号（从1开始）
	AudioResourceID string     `bson:"audio_resource_id" json:"audio_resource_id"` // 音频文件的 resource_id
	Duration        float64    `bson:"duration" json:"duration"`                   // 音频时长（秒）
	Text            string     `bson:"text" json:"text"`                           // 对应的解说文本
	Timestamps      []CharTime `bson:"timestamps" json:"timestamps"`               // 字符级别的时间戳
	Prompt          string     `bson:"prompt,omitempty" json:"prompt,omitempty"`   // 生成音频时使用的提示词/参数（TTS参数配置）
	Version         int        `bson:"version" json:"version"`                     // 版本号（用于支持多版本，默认 1）
	Status          TaskStatus `bson:"status" json:"status"`                       // 状态：pending, completed, failed
	CreatedAt       time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
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
			Keys:    bson.D{{Key: "narration_id", Value: 1}, {Key: "sequence", Value: 1}},
			Options: options.Index().SetName("idx_narration_sequence"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "workflow_id", Value: 1}},
			Options: options.Index().SetName("idx_workflow_id"),
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
			Keys:    bson.D{{Key: "narration_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_narration_version"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
