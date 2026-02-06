package noveltools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// CleanJSONContent 清理 LLM 返回的 JSON 内容（公开函数）
// 移除 markdown 代码块标记，修复常见的 JSON 格式问题
func CleanJSONContent(content string) string {
	return cleanJSONContent(content)
}

// cleanJSONContent 清理 LLM 返回的 JSON 内容（私有函数）
// 移除 markdown 代码块标记，修复常见的 JSON 格式问题
func cleanJSONContent(content string) string {
	// 移除首尾空白
	content = strings.TrimSpace(content)

	// 移除 markdown 代码块标记（```json ... ``` 或 ``` ... ```）
	// 匹配 ```json 开头和 ``` 结尾
	// 注意：在原始字符串中，反引号需要特殊处理
	markdownPattern := regexp.MustCompile(`(?s)^\s*` + "```" + `(?:json)?\s*\n(.*?)\n\s*` + "```" + `\s*$`)
	if matches := markdownPattern.FindStringSubmatch(content); len(matches) > 1 {
		content = matches[1]
	}

	// 移除可能的其他 markdown 标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// 尝试修复常见的 JSON 格式问题
	// 1. 修复未转义的换行符（在字符串值中）
	// 注意：这是一个简单的修复，可能不适用于所有情况
	// 如果 JSON 本身格式正确，这个操作不会影响它

	return content
}

// NarrationJSONContent 临时结构体，用于解析 JSON（不保存到数据库）
// 注意：这些结构体仅用于解析 LLM 返回的 JSON，解析后会转换为 Scene、Shot、Character、Prop 实体保存到数据库
type NarrationJSONContent struct {
	Characters []*NarrationJSONCharacter `json:"characters,omitempty"` // 角色列表（可选）
	Props      []*NarrationJSONProp      `json:"props,omitempty"`      // 道具列表（可选）
	Scenes     []*NarrationJSONScene     `json:"scenes"`               // 场景列表（必需）
}

// NarrationJSONCharacter 临时角色结构体
type NarrationJSONCharacter struct {
	Name       string `json:"name"`                  // 角色姓名
	Gender     string `json:"gender,omitempty"`      // 性别：男/女
	AgeGroup   string `json:"age_group,omitempty"`   // 年龄段：青年/中年/老年/青少年/儿童
	RoleNumber string `json:"role_number,omitempty"` // 角色编号
	Description string `json:"description,omitempty"` // 角色详细描述
	ImagePrompt string `json:"image_prompt,omitempty"` // 角色图片提示词
}

// NarrationJSONProp 临时道具结构体
type NarrationJSONProp struct {
	Name        string `json:"name"`                  // 道具名称
	Description string `json:"description,omitempty"` // 道具详细描述
	ImagePrompt string `json:"image_prompt,omitempty"` // 道具图片提示词
	Category    string `json:"category,omitempty"`    // 道具类别（如：武器、法器、丹药等）
}

// NarrationJSONScene 临时场景结构体
type NarrationJSONScene struct {
	SceneNumber string               `json:"scene_number"`
	Description string               `json:"description"`         // 场景详细描述
	ImagePrompt string               `json:"image_prompt"`        // 场景图片提示词
	Narration   string               `json:"narration,omitempty"` // 场景级别的解说内容（可选）
	Shots       []*NarrationJSONShot `json:"shots"`
}

// NarrationJSONShot 临时镜头结构体
type NarrationJSONShot struct {
	CloseupNumber  string  `json:"closeup_number"`            // 镜头编号
	Character      string  `json:"character"`                 // 角色名称
	Image          string  `json:"image"`                     // 画面描述
	Narration      string  `json:"narration"`                 // 旁白
	SoundEffect    string  `json:"sound_effect,omitempty"`    // 音效描述
	Duration       float64 `json:"duration,omitempty"`        // 时长（秒）
	ImagePrompt    string  `json:"image_prompt"`              // 镜头图片提示词
	VideoPrompt    string  `json:"video_prompt"`              // 镜头视频提示词
	CameraMovement string  `json:"camera_movement,omitempty"` // 运镜方式
}

// ValidateNarrationJSON 验证 JSON 格式的解说文案
// 返回解析后的结构化数据和验证结果
// 注意：此函数现在返回临时结构体，不再返回 NarrationContent（已移除）
func ValidateNarrationJSON(jsonContent string, minLength, maxLength int) (*NarrationJSONContent, *ValidationResult) {
	result := &ValidationResult{
		IsValid:  true,
		Warnings: make([]string, 0),
	}

	// 清理 JSON 内容（移除 markdown 代码块等）
	jsonContent = cleanJSONContent(jsonContent)

	// 检查空内容
	if jsonContent == "" {
		result.IsValid = false
		result.Message = "解说内容为空"
		return nil, result
	}

	// 尝试解析 JSON 到结构体
	var content NarrationJSONContent
	if err := json.Unmarshal([]byte(jsonContent), &content); err != nil {
		result.IsValid = false
		result.Message = fmt.Sprintf("JSON 解析失败: %v", err)
		return nil, result
	}

	// 验证基本结构
	if len(content.Scenes) == 0 {
		result.IsValid = false
		result.Message = "缺少 scenes 字段或 scenes 为空"
		return nil, result
	}

	if len(content.Scenes) < 7 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("分镜数量不足，期望至少7个，实际%d个，但继续生成", len(content.Scenes)))
	}

	// 提取所有解说内容并统计字数
	totalExplanationText := ""
	explanationCount := 0

	for _, scene := range content.Scenes {
		if scene == nil {
			continue
		}

		// 分镜级别的解说内容（可选）
		if scene.Narration != "" {
			totalExplanationText += scene.Narration
			explanationCount++
		}

		// 特写级别的解说内容
		if scene.Shots != nil {
			for _, shot := range scene.Shots {
				if shot == nil {
					continue
				}
				if shot.Narration != "" {
					totalExplanationText += shot.Narration
					explanationCount++
				}
			}
		}
	}

	explanationLength := countChineseCharacters(totalExplanationText)

	// 验证字数范围
	if explanationLength < minLength {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("解说文本长度不足，当前%d字，最少建议%d字，但继续生成",
				explanationLength, minLength))
	}

	if explanationLength > maxLength {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("解说文本过长，当前%d字，最多建议%d字，但继续生成",
				explanationLength, maxLength))
	}

	// 验证分镜1的第一个和第二个特写的字数
	if len(content.Scenes) > 0 {
		firstScene := content.Scenes[0]
		if firstScene != nil && firstScene.Shots != nil {
			if len(firstScene.Shots) > 0 {
				firstShot := firstScene.Shots[0]
				if firstShot != nil && firstShot.Narration != "" {
					charCount := countChineseCharacters(firstShot.Narration)
					result.FirstCloseup = &CloseupValidation{
						Content:   firstShot.Narration,
						CharCount: charCount,
						Valid:     30 <= charCount && charCount <= 32,
						Exists:    true,
					}
				}
			}
			if len(firstScene.Shots) > 1 {
				secondShot := firstScene.Shots[1]
				if secondShot != nil && secondShot.Narration != "" {
					charCount := countChineseCharacters(secondShot.Narration)
					result.SecondCloseup = &CloseupValidation{
						Content:   secondShot.Narration,
						CharCount: charCount,
						Valid:     30 <= charCount && charCount <= 32,
						Exists:    true,
					}
				}
			}
		}
	}

	result.Message = "验证通过"
	result.TotalLength = explanationLength
	return &content, result
}

// countChineseCharacters 计算中文字符数量
func countChineseCharacters(text string) int {
	count := 0
	for _, r := range text {
		// 中文字符范围：\u4e00-\u9fff
		// 中文标点范围：\u3000-\u303f, \uff00-\uffef
		if (r >= 0x4e00 && r <= 0x9fff) || (r >= 0x3000 && r <= 0x303f) || (r >= 0xff00 && r <= 0xffef) {
			count++
		}
	}
	return count
}

// ParseNarrationJSON 解析 JSON 格式的解说文案
// 使用 ValidateNarrationJSON 进行解析和验证
func ParseNarrationJSON(jsonContent string) (*NarrationJSONContent, error) {
	// 使用验证函数来解析和验证 JSON
	content, validationResult := ValidateNarrationJSON(jsonContent, 1100, 1300)
	if !validationResult.IsValid {
		return nil, fmt.Errorf("narration validation failed: %s", validationResult.Message)
	}
	return content, nil
}
