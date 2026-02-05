package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Character 角色实体（小说级别）
// 说明：角色信息在小说级别统一管理，所有章节共享
type Character struct {
	ID string `bson:"id" json:"id"` // 角色ID（UUID）

	NovelID    string `bson:"novel_id" json:"novel_id"`     // 关联的小说ID
	WorkflowID string `bson:"workflow_id" json:"workflow_id"` // 关联的工作流ID
	Name       string `bson:"name" json:"name"`              // 角色姓名

	Gender     string `bson:"gender,omitempty" json:"gender,omitempty"`           // 性别：男/女
	AgeGroup   string `bson:"age_group,omitempty" json:"age_group,omitempty"`     // 年龄段：青年/中年/老年/青少年/儿童
	RoleNumber string `bson:"role_number,omitempty" json:"role_number,omitempty"` // 角色编号

	Description string `bson:"description" json:"description"` // 角色详细描述
	ImagePrompt  string `bson:"image_prompt" json:"image_prompt"` // 角色图片提示词

	// Appearance 外貌特征
	Appearance *CharacterAppearance `bson:"appearance,omitempty" json:"appearance,omitempty"`

	// Clothing 服装风格
	Clothing *CharacterClothing `bson:"clothing,omitempty" json:"clothing,omitempty"`

	Status      TaskStatus `bson:"status" json:"status"`                           // 状态：pending, completed, failed
	ErrorMessage string    `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// CharacterAppearance 角色外貌特征
type CharacterAppearance struct {
	HairStyle string `bson:"hair_style,omitempty" json:"hair_style,omitempty"` // 发型
	HairColor string `bson:"hair_color,omitempty" json:"hair_color,omitempty"` // 发色
	Face      string `bson:"face,omitempty" json:"face,omitempty"`             // 面部特征
	Body      string `bson:"body,omitempty" json:"body,omitempty"`             // 身材特征
}

// CharacterClothing 角色服装风格
type CharacterClothing struct {
	Top       string `bson:"top,omitempty" json:"top,omitempty"`             // 上衣
	Bottom    string `bson:"bottom,omitempty" json:"bottom,omitempty"`       // 下装
	Accessory string `bson:"accessory,omitempty" json:"accessory,omitempty"` // 配饰
}

// Collection 返回集合名称
func (c *Character) Collection() string { return "characters" }

// EnsureIndexes 创建和维护索引
func (c *Character) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(c.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "workflow_id", Value: 1}},
			Options: options.Index().SetName("idx_workflow_id"),
		},
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}, {Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_novel_name_unique"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
