package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ImageType 图片类型
type ImageType string

const (
	ImageTypeCharacter ImageType = "character"  // 角色图片
	ImageTypeProp      ImageType = "prop"       // 道具图片
	ImageTypeShotFirst ImageType = "shot_first" // 镜头首图
	ImageTypeShotLast  ImageType = "shot_last"  // 镜头尾图
)

// CharacterImageSubtype 角色图片细分类
type CharacterImageSubtype string

const (
	CharacterImageSubtypeFront     CharacterImageSubtype = "front"      // 正视图
	CharacterImageSubtypeThreeView CharacterImageSubtype = "three_view" // 三视图
	CharacterImageSubtypeDetail    CharacterImageSubtype = "detail"     // 细节图
)

// Image 图片实体
// 说明：支持角色图、道具图、镜头首图、镜头尾图，每种类型可以有多版本（用于多次抽卡）
type Image struct {
	// 基础标识
	ID string `bson:"id" json:"id"` // 图片ID（UUID）

	// 关联字段
	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID

	// 类型和关联（根据 ImageType 确定哪个字段不为空）
	ImageType             ImageType             `bson:"image_type" json:"image_type"`                                               // 图片类型：character（角色图）、prop（道具图）、shot_first（镜头首图）、shot_last（镜头尾图）
	CharacterID           string                `bson:"character_id,omitempty" json:"character_id,omitempty"`                       // 关联的角色ID（角色图片时必填）
	CharacterImageSubtype CharacterImageSubtype `bson:"character_image_subtype,omitempty" json:"character_image_subtype,omitempty"` // 角色图片细分类：front（正视图）、three_view（三视图）、detail（细节图）
	PropID                string                `bson:"prop_id,omitempty" json:"prop_id,omitempty"`                                 // 关联的道具ID（道具图片时必填）
	ShotID                string                `bson:"shot_id,omitempty" json:"shot_id,omitempty"`                                 // 关联的镜头ID（镜头首图/尾图时必填）

	// 图片信息
	ImageResourceID string `bson:"image_resource_id" json:"image_resource_id"` // 图片文件的 resource_id
	Prompt          string `bson:"prompt" json:"prompt"`                       // 生成图片时使用的完整 prompt（保留用于多次抽卡）

	// 版本信息
	Version int `bson:"version" json:"version"` // 版本号（用于支持多版本抽卡，从1开始递增）

	// 状态信息
	Status       TaskStatus `bson:"status" json:"status"`                                   // 状态：pending, completed, failed
	ErrorMessage string     `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Collection 返回集合名称
func (i *Image) Collection() string { return "images" }

// EnsureIndexes 创建和维护索引
func (i *Image) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(i.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "novel_id", Value: 1}},
			Options: options.Index().SetName("idx_novel_id"),
		},
		{
			Keys:    bson.D{{Key: "image_type", Value: 1}},
			Options: options.Index().SetName("idx_image_type"),
		},
		{
			// 角色图片索引（支持多版本和细分类）
			Keys:    bson.D{{Key: "character_id", Value: 1}, {Key: "character_image_subtype", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"character_id": bson.M{"$exists": true, "$ne": ""}}).SetName("idx_character_subtype_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "character_image_subtype", Value: 1}},
			Options: options.Index().SetName("idx_character_image_subtype"),
		},
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetName("idx_character_id"),
		},
		{
			// 道具图片索引（支持多版本）
			Keys:    bson.D{{Key: "prop_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"prop_id": bson.M{"$exists": true, "$ne": ""}}).SetName("idx_prop_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "prop_id", Value: 1}},
			Options: options.Index().SetName("idx_prop_id"),
		},
		{
			// 镜头首图索引（支持多版本）
			Keys:    bson.D{{Key: "shot_id", Value: 1}, {Key: "image_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"shot_id": bson.M{"$exists": true, "$ne": ""}, "image_type": ImageTypeShotFirst}).SetName("idx_shot_first_version_unique"),
		},
		{
			// 镜头尾图索引（支持多版本）
			Keys:    bson.D{{Key: "shot_id", Value: 1}, {Key: "image_type", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"shot_id": bson.M{"$exists": true, "$ne": ""}, "image_type": ImageTypeShotLast}).SetName("idx_shot_last_version_unique"),
		},
		{
			Keys:    bson.D{{Key: "shot_id", Value: 1}},
			Options: options.Index().SetName("idx_shot_id"),
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
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
