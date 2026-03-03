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
	// 基础标识
	ID string `bson:"id" json:"id"` // 角色ID（UUID）

	// 关联字段
	NovelID string `bson:"novel_id" json:"novel_id"` // 关联的小说ID

	// 基本信息
	Name       string `bson:"name" json:"name"`                                   // 角色名称（对应 character.md 中的 name）
	Gender     string `bson:"gender,omitempty" json:"gender,omitempty"`           // 性别（便于兼容旧结构，优先使用 BaseProfile.Gender）
	AgeGroup   string `bson:"age_group,omitempty" json:"age_group,omitempty"`     // 年龄段（便于兼容旧结构，优先使用 BaseProfile.AgeRange）
	RoleNumber string `bson:"role_number,omitempty" json:"role_number,omitempty"` // 角色编号（兼容旧结构）

	// 资产结构（对应 character.md 中的 JSON 结构）
	PriorityLevel string                   `bson:"priority_level,omitempty" json:"priority_level,omitempty"` // 推荐等级：S/A/B/C
	BaseProfile   *CharacterBaseProfile    `bson:"base_profile,omitempty" json:"base_profile,omitempty"`     // 角色本体层信息
	StateVersions []*CharacterStateVersion `bson:"state_versions,omitempty" json:"state_versions,omitempty"` // 角色状态版本列表
	Relationships *CharacterRelationships  `bson:"relationships,omitempty" json:"relationships,omitempty"`   // 角色关系标签

	// 描述信息（兼容字段）
	Description string `bson:"description,omitempty" json:"description,omitempty"`   // 角色详细描述（可由 BaseProfile 中信息汇总而来）
	ImagePrompt string `bson:"image_prompt,omitempty" json:"image_prompt,omitempty"` // 角色图片提示词（历史字段，推荐用 Prompt 模块生成）

	// 关联图片
	ImageID string `bson:"image_id,omitempty" json:"image_id,omitempty"` // 当前绑定的角色图片ID（用户选择的版本）

	// 嵌套结构
	Appearance *CharacterAppearance `bson:"appearance,omitempty" json:"appearance,omitempty"` // 外貌特征
	Clothing   *CharacterClothing   `bson:"clothing,omitempty" json:"clothing,omitempty"`     // 服装风格

	// 状态信息
	Status       TaskStatus `bson:"status" json:"status"`                                   // 状态：pending, completed, failed
	ErrorMessage string     `bson:"error_message,omitempty" json:"error_message,omitempty"` // 错误信息（失败时）

	// 时间戳
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// CharacterAppearance 角色外貌特征
type CharacterAppearance struct {
	FaceShape      string `bson:"face_shape,omitempty" json:"face_shape,omitempty"`           // 脸型
	FacialFeatures string `bson:"facial_features,omitempty" json:"facial_features,omitempty"` // 五官特征
	EyeColor       string `bson:"eye_color,omitempty" json:"eye_color,omitempty"`             // 眼睛颜色
	HairStyle      string `bson:"hair_style,omitempty" json:"hair_style,omitempty"`           // 发型
	HairColor      string `bson:"hair_color,omitempty" json:"hair_color,omitempty"`           // 发色
	SkinTone       string `bson:"skin_tone,omitempty" json:"skin_tone,omitempty"`             // 肤色
	SpecialMarks   string `bson:"special_marks,omitempty" json:"special_marks,omitempty"`     // 特征标志（疤痕/纹身等）
	Posture        string `bson:"posture,omitempty" json:"posture,omitempty"`                 // 体态特征（站姿/走路习惯）
}

// CharacterClothing 角色服装风格
type CharacterClothing struct {
	CommonType    string `bson:"common_type,omitempty" json:"common_type,omitempty"`       // 常见服装类型
	ColorPalette  string `bson:"color_palette,omitempty" json:"color_palette,omitempty"`   // 常见配色
	MaterialStyle string `bson:"material_style,omitempty" json:"material_style,omitempty"` // 材质倾向
	EraSetting    string `bson:"era_setting,omitempty" json:"era_setting,omitempty"`       // 时代定位
}

// CharacterBaseProfile 角色本体层（长期固定资产）
type CharacterBaseProfile struct {
	Gender          string               `bson:"gender,omitempty" json:"gender,omitempty"`                     // 性别
	AgeRange        string               `bson:"age_range,omitempty" json:"age_range,omitempty"`               // 年龄段
	Height          string               `bson:"height,omitempty" json:"height,omitempty"`                     // 身高
	BodyType        string               `bson:"body_type,omitempty" json:"body_type,omitempty"`               // 体型
	Identity        string               `bson:"identity,omitempty" json:"identity,omitempty"`                 // 种族/身份
	SocialStatus    string               `bson:"social_status,omitempty" json:"social_status,omitempty"`       // 社会地位
	FirstAppearance string               `bson:"first_appearance,omitempty" json:"first_appearance,omitempty"` // 初次登场章节
	Appearance      *CharacterAppearance `bson:"appearance,omitempty" json:"appearance,omitempty"`             // 外貌物理特征
	Clothing        *CharacterClothing   `bson:"clothing,omitempty" json:"clothing,omitempty"`                 // 服装基础风格

	VisualPersonalityKeywords []string                    `bson:"visual_personality_keywords,omitempty" json:"visual_personality_keywords,omitempty"` // 核心性格视觉关键词
	SignatureElements         *CharacterSignatureElements `bson:"signature_elements,omitempty" json:"signature_elements,omitempty"`                   // 标志性元素
}

// CharacterSignatureElements 标志性元素
type CharacterSignatureElements struct {
	Weapon            string `bson:"weapon,omitempty" json:"weapon,omitempty"`                           // 武器
	Accessories       string `bson:"accessories,omitempty" json:"accessories,omitempty"`                 // 饰品
	Items             string `bson:"items,omitempty" json:"items,omitempty"`                             // 随身物品
	AbilityVisualForm string `bson:"ability_visual_form,omitempty" json:"ability_visual_form,omitempty"` // 能力的物理表现形式
}

// CharacterStateVersion 角色状态版本
type CharacterStateVersion struct {
	StateName         string `bson:"state_name,omitempty" json:"state_name,omitempty"`                 // 状态版本名称
	TriggerCondition  string `bson:"trigger_condition,omitempty" json:"trigger_condition,omitempty"`   // 触发条件
	AppearanceChange  string `bson:"appearance_change,omitempty" json:"appearance_change,omitempty"`   // 外貌变化
	ClothingChange    string `bson:"clothing_change,omitempty" json:"clothing_change,omitempty"`       // 服装变化
	TemperamentChange string `bson:"temperament_change,omitempty" json:"temperament_change,omitempty"` // 气质变化
	AbilityChange     string `bson:"ability_change,omitempty" json:"ability_change,omitempty"`         // 能力变化
	IsLongTerm        bool   `bson:"is_long_term,omitempty" json:"is_long_term,omitempty"`             // 是否长期存在
}

// CharacterRelationships 角色关系核心标签
type CharacterRelationships struct {
	CoreRelations      string `bson:"core_relations,omitempty" json:"core_relations,omitempty"`           // 与主要角色关系
	Faction            string `bson:"faction,omitempty" json:"faction,omitempty"`                         // 阵营归属
	EmotionalDirection string `bson:"emotional_direction,omitempty" json:"emotional_direction,omitempty"` // 情感方向
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
