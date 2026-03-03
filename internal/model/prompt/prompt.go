package prompt

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PromptTemplate 提示词模板实体
// 用于存储可配置的多种提示词模板（角色、场景、镜头等），支持关键字占位符替换。
type PromptTemplate struct {
	// 基础标识
	ID string `bson:"id" json:"id"`

	// 模板分类与编码
	// Type 例如：character、scene、shot、shot_video、character_image
	Type string `bson:"type" json:"type"`
	// Code 为同一 Type 下的业务编码，例如：default、anime_style 等
	Code string `bson:"code" json:"code"`

	// 语言标识（可选），例如：zh-CN、en-US
	Language string `bson:"language,omitempty" json:"language,omitempty"`

	// 管理信息
	Title       string `bson:"title,omitempty" json:"title,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`

	// 模板正文，包含占位符（例如：{{CHARACTER_NAME}}）
	Content string `bson:"content" json:"content"`

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (p *PromptTemplate) Collection() string {
	return "prompt_templates"
}

// EnsureIndexes 创建和维护索引
// 为 type + code + language 创建唯一索引，便于按业务维度快速查找模板。
func (p *PromptTemplate) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(p.Collection())

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "code", Value: 1},
				{Key: "language", Value: 1},
			},
			Options: options.Index().SetName("idx_type_code_lang").SetUnique(true),
		},
	}

	if len(indexes) == 0 {
		return nil
	}

	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}

