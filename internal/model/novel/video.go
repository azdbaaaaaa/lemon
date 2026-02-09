package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VideoStatus 视频状态（包含 processing 状态）
type VideoStatus string

const (
	VideoStatusPending    VideoStatus = "pending"    // 待处理
	VideoStatusProcessing VideoStatus = "processing" // 处理中
	VideoStatusCompleted  VideoStatus = "completed"  // 已完成
	VideoStatusFailed     VideoStatus = "failed"     // 失败
)

// String 返回状态的字符串表示
func (s VideoStatus) String() string {
	return string(s)
}

// VideoType 视频类型
type VideoType string

const (
	VideoTypeShot  VideoType = "shot"  // 分镜视频（关联镜头ID）
	VideoTypeNovel VideoType = "novel" // 完整视频（关联剧本ID）
)

// String 返回类型的字符串表示
func (t VideoType) String() string {
	return string(t)
}

// Video 视频实体
// 说明：支持两种类型：分镜视频（关联镜头ID）和完整视频（关联剧本ID）
type Video struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 视频ID（UUID）

	// 关联字段
	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID（两种类型都需要）
	UserID  string `bson:"user_id" json:"user_id"`   // 用户ID

	// 类型和关联（根据 VideoType 确定哪个字段不为空）
	VideoType VideoType `bson:"video_type" json:"video_type"`                     // 视频类型：shot（分镜视频）、novel（完整视频）
	ShotID    string    `bson:"shot_id,omitempty" json:"shot_id,omitempty"`       // 关联的镜头ID（分镜视频时必填）
	ChapterID string    `bson:"chapter_id,omitempty" json:"chapter_id,omitempty"` // 关联的章节ID（分镜视频时可选，用于查询）

	// 内容信息
	VideoResourceID string  `bson:"video_resource_id" json:"video_resource_id"` // 视频文件的 resource_id
	Duration        float64 `bson:"duration" json:"duration"`                   // 视频时长（秒）
	Prompt          string  `bson:"prompt,omitempty" json:"prompt,omitempty"`   // 生成视频时使用的提示词/参数

	// 版本信息
	Version int `bson:"version" json:"version"` // 版本号（用于支持多版本，默认 1）

	// 状态信息
	Status       VideoStatus `bson:"status" json:"status"`                                   // 状态：pending, processing, completed, failed
	ErrorMessage string      `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
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
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "video_type", Value: 1}},
			Options: options.Index().SetName("idx_video_type"),
		},
		{
			// 分镜视频索引（支持多版本）
			Keys:    bson.D{{Key: "shot_id", Value: 1}, {Key: "video_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"shot_id": bson.M{"$exists": true, "$type": "string"}, "video_type": VideoTypeShot}).SetName("idx_shot_type_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "shot_id", Value: 1}},
			Options: options.Index().SetName("idx_shot_id"),
		},
		{
			// 完整视频索引（支持多版本）
			Keys:    bson.D{{Key: "novel_id", Value: 1}, {Key: "video_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"video_type": VideoTypeNovel}).SetName("idx_novel_type_version_unique"),
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
