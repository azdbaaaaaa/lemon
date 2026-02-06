package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Prop 道具实体（小说级别）
// 说明：道具信息在小说级别统一管理，所有章节共享
type Prop struct {
	ID string `bson:"id" json:"id"` // 道具ID（UUID）

	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID
	Name    string `bson:"name" json:"name"`         // 道具名称

	Description     string `bson:"description" json:"description"`           // 道具详细描述
	ImagePrompt     string `bson:"image_prompt" json:"image_prompt"`          // 道具图片提示词
	ImageResourceID string `bson:"image_resource_id,omitempty" json:"image_resource_id,omitempty"` // 道具图片的 resource_id

	Category string `bson:"category,omitempty" json:"category,omitempty"` // 道具类别（如：武器、法器、丹药等）

	Status      TaskStatus `bson:"status" json:"status"`                           // 状态：pending, completed, failed
	ErrorMessage string    `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (p *Prop) Collection() string { return "props" }

// EnsureIndexes 创建和维护索引
func (p *Prop) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(p.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}, {Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_novel_name_unique"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
