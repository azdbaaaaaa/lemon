package service

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"lemon/internal/model/prompt"
	"lemon/internal/pkg/id"
	promptrepo "lemon/internal/repository/prompt"
)

// PromptService 提示词服务接口
// 负责提示词模板的管理和基于结构体数据的模板渲染。
type PromptService interface {
	// CreateTemplate 创建提示词模板
	CreateTemplate(ctx context.Context, tpl *prompt.PromptTemplate) (string, error)
	// UpdateTemplate 更新提示词模板
	UpdateTemplate(ctx context.Context, id string, updates *UpdatePromptTemplateRequest) error
	// GetTemplate 根据 ID 获取提示词模板
	GetTemplate(ctx context.Context, id string) (*prompt.PromptTemplate, error)
	// GetTemplateByTypeCode 根据类型、编码、语言获取提示词模板
	GetTemplateByTypeCode(ctx context.Context, promptType, code, language string) (*prompt.PromptTemplate, error)
	// RenderPrompt 渲染提示词
	// promptType 例如：character、scene、shot、shot_video、character_image
	// vars 为占位符映射，例如 {"CHARACTER_NAME": "张三"}
	RenderPrompt(ctx context.Context, promptType, code, language string, vars map[string]string) (string, error)
}

// promptService 提示词服务实现
type promptService struct {
	repo promptrepo.PromptRepository
}

// NewPromptService 创建提示词服务
// 只需要传入 Mongo 数据库依赖，Repository 在内部创建。
func NewPromptService(db *mongo.Database) (PromptService, error) {
	repo := promptrepo.NewPromptRepository(db)
	return &promptService{
		repo: repo,
	}, nil
}

// UpdatePromptTemplateRequest 更新模板请求
type UpdatePromptTemplateRequest struct {
	Title       *string
	Description *string
	Content     *string
	Language    *string
}

// CreateTemplate 创建提示词模板
func (s *promptService) CreateTemplate(ctx context.Context, tpl *prompt.PromptTemplate) (string, error) {
	tpl.ID = id.New()
	if err := s.repo.Create(ctx, tpl); err != nil {
		return "", err
	}
	return tpl.ID, nil
}

// UpdateTemplate 更新提示词模板
func (s *promptService) UpdateTemplate(ctx context.Context, id string, req *UpdatePromptTemplateRequest) error {
	updates := bson.M{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	if len(updates) == 0 {
		return nil
	}

	return s.repo.Update(ctx, id, updates)
}

// GetTemplate 根据 ID 获取提示词模板
func (s *promptService) GetTemplate(ctx context.Context, id string) (*prompt.PromptTemplate, error) {
	return s.repo.FindByID(ctx, id)
}

// GetTemplateByTypeCode 根据类型、编码、语言获取提示词模板
func (s *promptService) GetTemplateByTypeCode(ctx context.Context, promptType, code, language string) (*prompt.PromptTemplate, error) {
	return s.repo.FindByTypeCode(ctx, promptType, code, language)
}

// CharacterPromptData 角色提示词渲染数据
type CharacterPromptData struct {
	// 这里的字段命名与占位符一一对应，方便映射
	CharacterName string
	Gender        string
	AgeRange      string
	BodyType      string

	FaceShape      string
	FacialFeatures string
	EyeColor       string
	HairStyle      string
	HairColor      string
	SkinTone       string
	SpecialMarks   string

	ClothingType  string
	ColorPalette  string
	MaterialStyle string
	EraSetting    string
	SignatureItem string
	Temperament   string
}

// ScenePromptData 场景提示词渲染数据
type ScenePromptData struct {
	ChapterContent string
	ExistingScenes string
}

// ShotPromptData 镜头结构提示词渲染数据
type ShotPromptData struct {
	SceneJSON            string
	SceneExcerpt         string
	CharacterConstraints string
}

// ShotVideoPromptData 镜头视频提示词渲染数据
type ShotVideoPromptData struct {
	ShotJSON         string
	CharacterLibrary string
	SceneLibrary     string
}

// CharacterImagePromptData 角色图像提示词渲染数据
type CharacterImagePromptData struct {
	CharacterName       string
	Gender              string
	AgeRange            string
	BodyType            string
	FaceShape           string
	FacialFeatures      string
	EyeColor            string
	HairStyle           string
	HairColor           string
	SkinTone            string
	SpecialMarks        string
	ClothingType        string
	ColorPalette        string
	MaterialStyle       string
	EraSetting          string
	SignatureItem       string
	TemperamentKeywords string
}

// RenderPrompt 渲染提示词
func (s *promptService) RenderPrompt(ctx context.Context, promptType, code, language string, vars map[string]string) (string, error) {
	tpl, err := s.repo.FindByTypeCode(ctx, promptType, code, language)
	if err != nil {
		return "", err
	}

	return renderTemplate(tpl.Content, vars), nil
}

// buildPromptVars 根据提示词类型和数据结构构建占位符变量映射
func buildPromptVars(promptType string, data interface{}) (map[string]string, error) {
	switch promptType {
	case "character":
		d, ok := data.(*CharacterPromptData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for character prompt: %T", data)
		}
		return buildCharacterVars(d), nil
	case "scene":
		d, ok := data.(*ScenePromptData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for scene prompt: %T", data)
		}
		return buildSceneVars(d), nil
	case "shot":
		d, ok := data.(*ShotPromptData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for shot prompt: %T", data)
		}
		return buildShotVars(d), nil
	case "shot_video":
		d, ok := data.(*ShotVideoPromptData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for shot_video prompt: %T", data)
		}
		return buildShotVideoVars(d), nil
	case "character_image":
		d, ok := data.(*CharacterImagePromptData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for character_image prompt: %T", data)
		}
		return buildCharacterImageVars(d), nil
	default:
		return nil, fmt.Errorf("unsupported prompt type: %s", promptType)
	}
}

func buildCharacterVars(data *CharacterPromptData) map[string]string {
	return map[string]string{
		"CHARACTER_NAME": tplSafe(data.CharacterName),
		"GENDER":         tplSafe(data.Gender),
		"AGE_RANGE":      tplSafe(data.AgeRange),
		"BODY_TYPE":      tplSafe(data.BodyType),

		"FACE_SHAPE":      tplSafe(data.FaceShape),
		"FACIAL_FEATURES": tplSafe(data.FacialFeatures),
		"EYE_COLOR":       tplSafe(data.EyeColor),
		"HAIR_STYLE":      tplSafe(data.HairStyle),
		"HAIR_COLOR":      tplSafe(data.HairColor),
		"SKIN_TONE":       tplSafe(data.SkinTone),
		"SPECIAL_MARKS":   tplSafe(data.SpecialMarks),

		"CLOTHING_TYPE":  tplSafe(data.ClothingType),
		"COLOR_PALETTE":  tplSafe(data.ColorPalette),
		"MATERIAL_STYLE": tplSafe(data.MaterialStyle),
		"ERA_SETTING":    tplSafe(data.EraSetting),

		"SIGNATURE_ITEM":       tplSafe(data.SignatureItem),
		"TEMPERAMENT_KEYWORDS": tplSafe(data.Temperament),
	}
}

func buildSceneVars(data *ScenePromptData) map[string]string {
	return map[string]string{
		"CHAPTER_CONTENT": tplSafe(data.ChapterContent),
		"EXISTING_SCENES": tplSafe(data.ExistingScenes),
	}
}

func buildShotVars(data *ShotPromptData) map[string]string {
	return map[string]string{
		"SCENE_JSON":            tplSafe(data.SceneJSON),
		"SCENE_EXCERPT":         tplSafe(data.SceneExcerpt),
		"CHARACTER_CONSTRAINTS": tplSafe(data.CharacterConstraints),
	}
}

func buildShotVideoVars(data *ShotVideoPromptData) map[string]string {
	return map[string]string{
		"SHOT_JSON":         tplSafe(data.ShotJSON),
		"CHARACTER_LIBRARY": tplSafe(data.CharacterLibrary),
		"SCENE_LIBRARY":     tplSafe(data.SceneLibrary),
	}
}

func buildCharacterImageVars(data *CharacterImagePromptData) map[string]string {
	return map[string]string{
		"CHARACTER_NAME":       tplSafe(data.CharacterName),
		"GENDER":               tplSafe(data.Gender),
		"AGE_RANGE":            tplSafe(data.AgeRange),
		"BODY_TYPE":            tplSafe(data.BodyType),
		"FACE_SHAPE":           tplSafe(data.FaceShape),
		"FACIAL_FEATURES":      tplSafe(data.FacialFeatures),
		"EYE_COLOR":            tplSafe(data.EyeColor),
		"HAIR_STYLE":           tplSafe(data.HairStyle),
		"HAIR_COLOR":           tplSafe(data.HairColor),
		"SKIN_TONE":            tplSafe(data.SkinTone),
		"SPECIAL_MARKS":        tplSafe(data.SpecialMarks),
		"CLOTHING_TYPE":        tplSafe(data.ClothingType),
		"COLOR_PALETTE":        tplSafe(data.ColorPalette),
		"MATERIAL_STYLE":       tplSafe(data.MaterialStyle),
		"ERA_SETTING":          tplSafe(data.EraSetting),
		"SIGNATURE_ITEM":       tplSafe(data.SignatureItem),
		"TEMPERAMENT_KEYWORDS": tplSafe(data.TemperamentKeywords),
	}
}

// renderTemplate 执行 {{KEY}} 占位符替换
// 这里只做简单的字符串替换，不执行任何模板逻辑。
func renderTemplate(content string, vars map[string]string) string {
	result := content
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// tplSafe 将空指针或空字符串统一转换，避免出现 <nil> 等异常内容。
func tplSafe(s string) string {
	return s
}
