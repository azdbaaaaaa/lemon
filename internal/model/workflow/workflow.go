package workflow

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WorkflowStatus 工作流状态
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusPaused    WorkflowStatus = "paused"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// WorkflowStage 工作流阶段
type WorkflowStage string

const (
	WorkflowStageScript     WorkflowStage = "script"
	WorkflowStageAsset      WorkflowStage = "asset"
	WorkflowStageStoryboard WorkflowStage = "storyboard"
	WorkflowStageAnimatic   WorkflowStage = "animatic"
	WorkflowStageVideo      WorkflowStage = "video"
	WorkflowStageEdit       WorkflowStage = "edit"
)

// NarrationType 旁白类型
type NarrationType string

const (
	NarrationTypeNarration NarrationType = "narration" // 旁白（解说）类型
	NarrationTypeDialogue  NarrationType = "dialogue"  // 真人对话类型
)

// Workflow 工作流实体
// 一个 Workflow 表示一次完整的视频创作流程
type Workflow struct {
	ID            string         `bson:"id" json:"id"`                         // 工作流ID（UUID）
	UserID        string         `bson:"user_id" json:"user_id"`               // 用户ID
	Name          string         `bson:"name" json:"name"`                     // 工作流名称
	InputType     string         `bson:"input_type" json:"input_type"`         // 输入类型：text, file
	ResourceID    string         `bson:"resource_id,omitempty" json:"resource_id,omitempty"`     // 关联的资源ID（文件模式）
	TextContent   string         `bson:"text_content,omitempty" json:"text_content,omitempty"`   // 纯文本模式下的原始内容（可选）
	NarrationType NarrationType  `bson:"narration_type" json:"narration_type"` // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
	Status        WorkflowStatus `bson:"status" json:"status"`                 // 状态
	CurrentStage  WorkflowStage  `bson:"current_stage" json:"current_stage"`   // 当前阶段
	Progress      float64        `bson:"progress" json:"progress"`             // 总体进度（0.0-1.0）
	CreatedAt     time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `bson:"updated_at" json:"updated_at"`
	CompletedAt   *time.Time     `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}

// Collection 返回集合名称
func (w *Workflow) Collection() string { return "workflows" }

// EnsureIndexes 创建和维护索引
func (w *Workflow) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(w.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}


