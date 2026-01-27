package narration

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Narration 解说文案实体
// 参考: docs/design/workflow/NARRATION_MODULE_DESIGN.md
// Narration模块负责将原始输入转换为可用于视频制作的解说文案
type Narration struct {
	ID         string `bson:"id" json:"id"`                   // 解说文案ID（UUID）
	WorkflowID string `bson:"workflow_id" json:"workflow_id"` // 所属工作流ID
	UserID     string `bson:"user_id" json:"user_id"`         // 所属用户ID

	// 输入资源（原始文件）
	InputResourceID string `bson:"input_resource_id,omitempty" json:"input_resource_id,omitempty"` // 输入文件资源ID（指向原始小说/文档资源）

	// 输出资源（生成的JSON文件）
	OutputResourceID string `bson:"output_resource_id,omitempty" json:"output_resource_id,omitempty"` // 输出文件资源ID（指向生成的解说文案JSON文件资源）

	// 版本和元数据
	Version  string             `bson:"version" json:"version"`                       // JSON格式版本号，格式：major.minor
	Metadata *NarrationMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"` // 元数据

	// 章节数据
	Chapters []*Chapter `bson:"chapters" json:"chapters"` // 章节数组

	// 验证报告
	ValidationReport *ValidationReport `bson:"validation_report,omitempty" json:"validation_report,omitempty"` // 验证报告（在验证后添加）

	// 状态
	Status NarrationStatus `bson:"status" json:"status"` // 状态

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`                     // 创建时间
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`                     // 更新时间
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"` // 软删除时间
}

// NarrationStatus 解说文案状态
type NarrationStatus string

const (
	NarrationStatusPending    NarrationStatus = "pending"    // 待生成
	NarrationStatusGenerating NarrationStatus = "generating" // 生成中
	NarrationStatusCompleted  NarrationStatus = "completed"  // 已完成
	NarrationStatusValidating NarrationStatus = "validating" // 验证中
	NarrationStatusValidated  NarrationStatus = "validated"  // 已验证
	NarrationStatusFailed     NarrationStatus = "failed"     // 失败
)

// NarrationMetadata 解说文案元数据
type NarrationMetadata struct {
	Title         string    `bson:"title,omitempty" json:"title,omitempty"`                   // 标题
	Author        string    `bson:"author,omitempty" json:"author,omitempty"`                 // 作者名
	GeneratedAt   time.Time `bson:"generated_at,omitempty" json:"generated_at,omitempty"`     // 生成时间
	GeneratedBy   string    `bson:"generated_by,omitempty" json:"generated_by,omitempty"`     // 生成服务名称，默认：narration_service
	AIProvider    string    `bson:"ai_provider,omitempty" json:"ai_provider,omitempty"`       // AI服务提供商（gemini/doubao/openai）
	AIModel       string    `bson:"ai_model,omitempty" json:"ai_model,omitempty"`             // AI模型名称
	TotalChapters int       `bson:"total_chapters,omitempty" json:"total_chapters,omitempty"` // 总章节数
	TotalShots    int       `bson:"total_shots,omitempty" json:"total_shots,omitempty"`       // 总分镜数
	Status        string    `bson:"status,omitempty" json:"status,omitempty"`                 // 状态（pending/generating/completed/validating/validated/failed）
}

// Chapter 章节实体
type Chapter struct {
	ID       string           `bson:"id" json:"id"`                                 // 章节唯一标识，格式：chapter_{identifier}
	Sequence int              `bson:"sequence" json:"sequence"`                     // 章节序号，从1开始，必须连续
	Title    string           `bson:"title" json:"title"`                           // 章节标题
	Metadata *ChapterMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"` // 章节元数据
	Shots    []*Shot          `bson:"shots" json:"shots"`                           // 分镜数组，至少包含一个分镜
}

// ChapterMetadata 章节元数据
type ChapterMetadata struct {
	OriginalTitle string `bson:"original_title,omitempty" json:"original_title,omitempty"` // 原始章节标题
	WordCount     int    `bson:"word_count,omitempty" json:"word_count,omitempty"`         // 章节总字数
	ShotCount     int    `bson:"shot_count,omitempty" json:"shot_count,omitempty"`         // 章节分镜数
}

// Shot 分镜实体
type Shot struct {
	ID        string        `bson:"id" json:"id"`                                 // 分镜唯一标识，格式：shot_{identifier}
	Sequence  int           `bson:"sequence" json:"sequence"`                     // 分镜序号，在章节内从1开始，必须连续
	Scene     string        `bson:"scene" json:"scene"`                           // 场景描述
	Narration string        `bson:"narration" json:"narration"`                   // 解说文案，字数应在20-100字之间（验证后）
	Dialogue  *string       `bson:"dialogue,omitempty" json:"dialogue,omitempty"` // 台词，可选，null表示无台词
	ShotType  string        `bson:"shot_type" json:"shot_type"`                   // 景别类型，枚举值: "close-up", "medium", "wide"
	Metadata  *ShotMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"` // 分镜元数据
}

// ShotType 景别类型
type ShotType string

const (
	ShotTypeCloseUp ShotType = "close-up" // 特写
	ShotTypeMedium  ShotType = "medium"   // 中景
	ShotTypeWide    ShotType = "wide"     // 全景
)

// ShotMetadata 分镜元数据
type ShotMetadata struct {
	WordCount        int        `bson:"word_count,omitempty" json:"word_count,omitempty"`               // 字数统计
	DurationEstimate float64    `bson:"duration_estimate,omitempty" json:"duration_estimate,omitempty"` // 预估时长（秒）
	Status           string     `bson:"status,omitempty" json:"status,omitempty"`                       // 验证状态，枚举值: "valid", "too_short", "too_long", "invalid"
	ValidatedAt      *time.Time `bson:"validated_at,omitempty" json:"validated_at,omitempty"`           // 验证时间
	Fixed            bool       `bson:"fixed,omitempty" json:"fixed,omitempty"`                         // 是否已修复
	FixType          string     `bson:"fix_type,omitempty" json:"fix_type,omitempty"`                   // 修复类型，枚举值: "shorten", "expand"
}

// ShotValidationStatus 分镜验证状态
type ShotValidationStatus string

const (
	ShotValidationStatusValid    ShotValidationStatus = "valid"     // 符合要求
	ShotValidationStatusTooShort ShotValidationStatus = "too_short" // 字数过少
	ShotValidationStatusTooLong  ShotValidationStatus = "too_long"  // 字数过多
	ShotValidationStatusInvalid  ShotValidationStatus = "invalid"   // 严重异常
)

// FixType 修复类型
type FixType string

const (
	FixTypeShorten FixType = "shorten" // 精简
	FixTypeExpand  FixType = "expand"  // 扩展
)

// ValidationReport 验证报告
type ValidationReport struct {
	ValidatedAt  time.Time             `bson:"validated_at" json:"validated_at"`                     // 验证时间
	TotalShots   int                   `bson:"total_shots" json:"total_shots"`                       // 总分镜数
	ValidShots   int                   `bson:"valid_shots" json:"valid_shots"`                       // 符合要求的分镜数
	InvalidShots int                   `bson:"invalid_shots" json:"invalid_shots"`                   // 不符合要求的分镜数
	FixedShots   int                   `bson:"fixed_shots,omitempty" json:"fixed_shots,omitempty"`   // 修复成功数
	FailedShots  int                   `bson:"failed_shots,omitempty" json:"failed_shots,omitempty"` // 修复失败数
	Statistics   *ValidationStatistics `bson:"statistics,omitempty" json:"statistics,omitempty"`     // 统计信息
}

// ValidationStatistics 验证统计信息
type ValidationStatistics struct {
	MinWords     int     `bson:"min_words,omitempty" json:"min_words,omitempty"`         // 最小字数要求
	MaxWords     int     `bson:"max_words,omitempty" json:"max_words,omitempty"`         // 最大字数要求
	AverageWords float64 `bson:"average_words,omitempty" json:"average_words,omitempty"` // 平均字数
	MedianWords  int     `bson:"median_words,omitempty" json:"median_words,omitempty"`   // 中位数字数
}

// Collection 返回集合名称
func (n *Narration) Collection() string {
	return "narrations"
}

// EnsureIndexes 创建和维护索引
// 参考: docs/design/workflow/NARRATION_MODULE_DESIGN.md
func (n *Narration) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(n.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "workflow_id", Value: 1}},
			Options: options.Index().SetName("idx_workflow_id"),
		},
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}, bson.E{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_user_status"),
		},
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}, bson.E{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{bson.E{Key: "input_resource_id", Value: 1}},
			Options: options.Index().SetName("idx_input_resource_id"),
		},
		{
			Keys:    bson.D{bson.E{Key: "output_resource_id", Value: 1}},
			Options: options.Index().SetName("idx_output_resource_id"),
		},
		{
			Keys:    bson.D{bson.E{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	}

	if len(indexes) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
