package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskStatus 任务状态（用于 Scene, Shot, Character, Prop, Image, Audio, Subtitle）
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待处理
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
)

// String 返回状态的字符串表示
func (s TaskStatus) String() string {
	return string(s)
}

// Scene 场景实体
// 说明：场景按章节存储，每个章节可以有多个场景，通过 sequence 标识顺序
type Scene struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 场景ID（UUID）

	// 关联字段
	ChapterID string `bson:"chapter_id" json:"chapter_id"` // 关联的章节ID
	NovelID   string `bson:"novel_id" json:"novel_id"`     // 关联的小说ID
	UserID    string `bson:"user_id" json:"user_id"`       // 用户ID（冗余字段，方便查询）

	// 内容信息
	Description string `bson:"description" json:"description"` // 场景详细描述
	Sequence    int    `bson:"sequence" json:"sequence"`       // 序号（在章节中的顺序，从1开始）

	// 版本管理
	Version int `bson:"version" json:"version"` // 场景版本号（从1开始）

	// 状态信息
	Status       TaskStatus `bson:"status" json:"status"`                                   // 状态：pending, completed, failed
	ErrorMessage string     `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (s *Scene) Collection() string {
	return "scenes"
}

// EnsureIndexes 创建和维护索引
func (s *Scene) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(s.Collection())

	// 删除旧的唯一索引（如果存在）
	// 从 (chapter_id, sequence) 迁移到 (chapter_id, version, sequence)
	oldIndexName := "idx_chapter_sequence_unique"
	_, err := coll.Indexes().DropOne(ctx, oldIndexName)
	if err != nil && err != mongo.ErrNoDocuments {
		// 忽略索引不存在的错误，其他错误记录但不阻止继续
		// 注意：MongoDB 驱动可能返回不同的错误类型，这里只处理常见的
	}

	indexes := []mongo.IndexModel{
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
