package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChapterNarration 章节解说实体
// 说明：章节解说单独存储，通过 chapter_id 关联章节
// Content 字段使用 NarrationContent 结构体存储结构化数据
type ChapterNarration struct {
	ID        string            `bson:"id" json:"id"`                               // 章节解说ID（UUID）
	ChapterID string            `bson:"chapter_id" json:"chapter_id"`               // 关联的章节ID
	UserID    string            `bson:"user_id" json:"user_id"`                     // 用户ID
	Content   *NarrationContent `bson:"content" json:"content"`                     // 章节解说内容（结构化数据）
	Prompt    string            `bson:"prompt,omitempty" json:"prompt,omitempty"`   // 生成章节解说时使用的提示词
	Version   int               `bson:"version" json:"version"`                       // 版本号（用于支持多版本，默认 1）
	Status    string            `bson:"status" json:"status"`                         // 状态：pending, completed, failed
	CreatedAt time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time        `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// NarrationContent 章节解说内容结构
// 对应 LLM 生成的 JSON 格式
type NarrationContent struct {
	ChapterInfo *NarrationChapterInfo `json:"chapter_info" bson:"chapter_info"`
	Characters  []*NarrationCharacter `json:"characters" bson:"characters"`
	Scenes      []*NarrationScene     `json:"scenes" bson:"scenes"`
}

// NarrationChapterInfo 章节解说中的章节信息
// 用于描述章节解说对应的章节信息，避免与章节模块的 ChapterInfo 命名冲突
type NarrationChapterInfo struct {
	ChapterNumber int    `json:"chapter_number" bson:"chapter_number"`
	Format        string `json:"format,omitempty" bson:"format,omitempty"`           // 章节风格（如：双时代格式、单一时代格式）
	PaintStyle    string `json:"paint_style,omitempty" bson:"paint_style,omitempty"` // 绘画风格（如：写实风格）
}

// NarrationCharacter 章节解说中的角色信息
// 用于描述章节解说中的角色，避免与其他模块的 Character 命名冲突
type NarrationCharacter struct {
	Name       string `json:"name" bson:"name"`                                   // 角色姓名
	Gender     string `json:"gender,omitempty" bson:"gender,omitempty"`           // 性别：男/女
	AgeGroup   string `json:"age_group,omitempty" bson:"age_group,omitempty"`     // 年龄组：青年/中年/老年/青少年/儿童
	RoleNumber string `json:"role_number,omitempty" bson:"role_number,omitempty"` // 角色编号
}

// NarrationScene 章节解说中的分镜信息
// 用于描述章节解说中的分镜，避免与其他模块的 Scene 命名冲突
type NarrationScene struct {
	SceneNumber string           `json:"scene_number" bson:"scene_number"`               // 分镜编号
	Narration   string           `json:"narration,omitempty" bson:"narration,omitempty"` // 分镜级别的解说内容（可选）
	Shots       []*NarrationShot `json:"shots" bson:"shots"`                             // 特写列表
}

// NarrationShot 章节解说中的特写信息
// 用于描述章节解说中的特写，避免与其他模块的 Shot 命名冲突
type NarrationShot struct {
	CloseupNumber string `json:"closeup_number" bson:"closeup_number"`                 // 特写编号
	Character     string `json:"character,omitempty" bson:"character,omitempty"`       // 特写人物姓名
	Narration     string `json:"narration" bson:"narration"`                           // 特写解说内容
	ScenePrompt   string `json:"scene_prompt,omitempty" bson:"scene_prompt,omitempty"` // 图片prompt描述
}

// Collection 返回集合名称
func (c *ChapterNarration) Collection() string {
	return "chapter_narrations"
}

// EnsureIndexes 创建和维护索引
func (c *ChapterNarration) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(c.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}},
			Options: options.Index().SetName("idx_chapter_id"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			// 一个章节只能有一个活跃的章节解说（未删除的）
			Keys: bson.D{
				{Key: "chapter_id", Value: 1},
				{Key: "deleted_at", Value: 1},
			},
			Options: options.Index().SetName("idx_chapter_deleted"),
		},
		{
			Keys:    bson.D{{Key: "chapter_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetName("idx_chapter_version"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
