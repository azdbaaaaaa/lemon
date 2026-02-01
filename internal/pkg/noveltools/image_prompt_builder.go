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

	if character.Gender != "" {
		genderDesc := "男性"
		if character.Gender == "女" {
			genderDesc = "女性"
		}
		parts = append(parts, fmt.Sprintf("一位%s", genderDesc))
	}

	if character.Appearance != nil {
		if character.Appearance.Face != "" {
			parts = append(parts, character.Appearance.Face)
		}
		if character.Appearance.HairStyle != "" && character.Appearance.HairColor != "" {
			parts = append(parts, fmt.Sprintf("%s%s", character.Appearance.HairColor, character.Appearance.HairStyle))
		}
		if character.Appearance.Body != "" {
			parts = append(parts, character.Appearance.Body)
		}
	}

	if character.Clothing != nil {
		var clothingParts []string
		if character.Clothing.Top != "" {
			clothingParts = append(clothingParts, character.Clothing.Top)
		}
		if character.Clothing.Bottom != "" {
			clothingParts = append(clothingParts, character.Clothing.Bottom)
		}
		if character.Clothing.Accessory != "" && character.Clothing.Accessory != "无其他装饰" {
			clothingParts = append(clothingParts, character.Clothing.Accessory)
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
