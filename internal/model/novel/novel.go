package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NarrationType 旁白类型
type NarrationType string

const (
	NarrationTypeNarration NarrationType = "narration" // 旁白（解说）类型
	NarrationTypeDialogue  NarrationType = "dialogue"  // 真人对话类型
)

// NovelStyle 剧本风格
type NovelStyle string

const (
	NovelStyleAnime NovelStyle = "anime" // 漫剧（动画风格）
	NovelStyleLive  NovelStyle = "live"  // 真人剧（真人风格）
	NovelStyleMixed NovelStyle = "mixed" // 混合风格
)

// EpisodeDuration 每集时长
type EpisodeDuration string

const (
	EpisodeDurationAuto   EpisodeDuration = "auto"     // 自动
	EpisodeDuration3To5   EpisodeDuration = "3-5min"   // 3-5分钟
	EpisodeDuration5To10  EpisodeDuration = "5-10min"  // 5-10分钟
	EpisodeDuration10To20 EpisodeDuration = "10-20min" // 10-20分钟
	EpisodeDuration20To30 EpisodeDuration = "20-30min" // 20-30分钟
)

// GenerationStatus 内容生成状态
type GenerationStatus string

const (
	GenerationStatusNone       GenerationStatus = ""           // 未开始
	GenerationStatusPending    GenerationStatus = "pending"    // 待处理
	GenerationStatusProcessing GenerationStatus = "processing" // 处理中
	GenerationStatusCompleted  GenerationStatus = "completed"  // 已完成
	GenerationStatusFailed     GenerationStatus = "failed"     // 失败
)

// Novel 剧本实体（主表）
// 用途：关联上传资源（resource_id），作为整个创作流程的核心实体
type Novel struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 剧本ID（UUID）

	// 关联字段
	UserID     string `bson:"user_id" json:"user_id"`         // 用户ID
	ResourceID string `bson:"resource_id" json:"resource_id"` // 关联上传的原始资源

	// 基本信息
	Title       string `bson:"title,omitempty" json:"title,omitempty"`             // 剧本名称
	Author      string `bson:"author,omitempty" json:"author,omitempty"`           // 作者
	Description string `bson:"description,omitempty" json:"description,omitempty"` // 简介

	// 创作配置
	NarrationType   NarrationType   `bson:"narration_type" json:"narration_type"`     // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
	Style           NovelStyle      `bson:"style" json:"style"`                       // 风格：anime（漫剧）、live（真人剧）、mixed（混合）
	EpisodeCount    int             `bson:"episode_count" json:"episode_count"`       // 集数（章节数量）
	EpisodeDuration EpisodeDuration `bson:"episode_duration" json:"episode_duration"` // 每集时长：auto（自动）、3-5min、5-10min、10-20min、20-30min

	// 内容生成状态（用于异步任务）
	GenerationStatus   GenerationStatus `bson:"generation_status,omitempty" json:"generation_status,omitempty"`     // 生成状态：pending, processing, completed, failed
	GenerationProgress int              `bson:"generation_progress,omitempty" json:"generation_progress,omitempty"` // 生成进度：0-100
	GenerationMessage  string           `bson:"generation_message,omitempty" json:"generation_message,omitempty"`   // 当前步骤说明

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (n *Novel) Collection() string { return "novels" }

// EnsureIndexes 创建和维护索引
func (n *Novel) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(n.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "resource_id", Value: 1}},
			Options: options.Index().SetName("idx_resource_id"),
		},
		{
			Keys:    bson.D{{Key: "style", Value: 1}},
			Options: options.Index().SetName("idx_style"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
