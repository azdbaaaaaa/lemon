package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Video 视频实体
// 说明：每个章节可能生成多种类型的视频（narration 视频、最终完整视频）
type Video struct {
	ID              string     `bson:"id" json:"id"`                                           // 视频ID（UUID）
	ChapterID       string     `bson:"chapter_id" json:"chapter_id"`                           // 关联的章节ID
	NarrationID     string     `bson:"narration_id,omitempty" json:"narration_id,omitempty"`   // 关联的解说ID（可选，final_video 没有 narration_id）
	WorkflowID      string     `bson:"workflow_id" json:"workflow_id"`                         // 关联的工作流ID
	UserID          string     `bson:"user_id" json:"user_id"`                                 // 用户ID
	Sequence        int        `bson:"sequence" json:"sequence"`                               // 视频片段序号（从1开始）
	VideoResourceID string     `bson:"video_resource_id" json:"video_resource_id"`             // 视频文件的 resource_id
	Duration        float64    `bson:"duration" json:"duration"`                               // 视频时长（秒）
	VideoType       VideoType   `bson:"video_type" json:"video_type"`                           // 视频类型：narration_video, final_video
	Prompt          string      `bson:"prompt,omitempty" json:"prompt,omitempty"`               // 生成视频时使用的提示词/参数
	Version         int         `bson:"version" json:"version"`                                 // 版本号（用于支持多版本，默认 1）
	Status          VideoStatus `bson:"status" json:"status"`                                   // 状态：pending, processing, completed, failed
	ErrorMessage    string     `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息
	CreatedAt       time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (v *Video) Collection() string {
	return "videos"
}

// EnsureIndexes 创建和维护索引
func (v *Video) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(v.Collection())
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
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "video_type", Value: 1}},
			Options: options.Index().SetName("idx_chapter_video_type"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
