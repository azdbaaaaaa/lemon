package noveltools

import (
	"fmt"
	"strings"

	"lemon/internal/model/novel"
)

// ImagePromptBuilder 图片 prompt 构建器
type ImagePromptBuilder struct {
	stylePrompt string
}

// NewImagePromptBuilder 创建图片 prompt 构建器
func NewImagePromptBuilder() *ImagePromptBuilder {
	return &ImagePromptBuilder{
		stylePrompt: "画面风格是强调强烈线条、鲜明对比和现代感造型，色彩饱和，带有动态夸张与都市叙事视觉冲击力的国风漫画风格",
	}
}

// BuildCharacterDescription 构建角色描述
func (b *ImagePromptBuilder) BuildCharacterDescription(character *novel.Character) string {
	var parts []string

	// 性别（优先使用 BaseProfile 中的信息）
	gender := character.Gender
	if character.BaseProfile != nil && character.BaseProfile.Gender != "" {
		gender = character.BaseProfile.Gender
	}
	if gender != "" {
		genderDesc := "男性"
		if strings.Contains(gender, "女") {
			genderDesc = "女性"
		}
		parts = append(parts, fmt.Sprintf("一位%s", genderDesc))
	}

	// 外貌信息（优先使用 BaseProfile.Appearance）
	var appearance *novel.CharacterAppearance
	if character.BaseProfile != nil && character.BaseProfile.Appearance != nil {
		appearance = character.BaseProfile.Appearance
	} else if character.Appearance != nil {
		appearance = character.Appearance
	}

	if appearance != nil {
		if appearance.FaceShape != "" {
			parts = append(parts, appearance.FaceShape)
		}
		if appearance.FacialFeatures != "" {
			parts = append(parts, appearance.FacialFeatures)
		}
		if appearance.HairStyle != "" && appearance.HairColor != "" {
			parts = append(parts, fmt.Sprintf("%s%s", appearance.HairColor, appearance.HairStyle))
		}
		if appearance.SkinTone != "" {
			parts = append(parts, appearance.SkinTone)
		}
		if appearance.SpecialMarks != "" {
			parts = append(parts, appearance.SpecialMarks)
		}
		if appearance.Posture != "" {
			parts = append(parts, appearance.Posture)
		}
	}

	// 体型（从 BaseProfile.BodyType）
	if character.BaseProfile != nil && character.BaseProfile.BodyType != "" {
		parts = append(parts, character.BaseProfile.BodyType)
	}

	// 服装信息（优先使用 BaseProfile.Clothing）
	var clothing *novel.CharacterClothing
	if character.BaseProfile != nil && character.BaseProfile.Clothing != nil {
		clothing = character.BaseProfile.Clothing
	} else if character.Clothing != nil {
		clothing = character.Clothing
	}

	if clothing != nil {
		var clothingParts []string
		if clothing.CommonType != "" {
			clothingParts = append(clothingParts, clothing.CommonType)
		}
		if clothing.ColorPalette != "" {
			clothingParts = append(clothingParts, clothing.ColorPalette)
		}
		if clothing.MaterialStyle != "" {
			clothingParts = append(clothingParts, clothing.MaterialStyle)
		}
		if clothing.EraSetting != "" {
			clothingParts = append(clothingParts, clothing.EraSetting)
		}
		if len(clothingParts) > 0 {
			parts = append(parts, fmt.Sprintf("身着%s", strings.Join(clothingParts, ", ")))
		}
	}

	return strings.Join(parts, "，")
}

// BuildCompletePrompt 构建完整的图片 prompt
// 格式：风格描述。角色描述。场景描述
func (b *ImagePromptBuilder) BuildCompletePrompt(character *novel.Character, scenePrompt string) string {
	stylePart := b.stylePrompt
	characterPart := b.BuildCharacterDescription(character)
	scenePart := scenePrompt

	return fmt.Sprintf("%s。%s。%s", stylePart, characterPart, scenePart)
}
