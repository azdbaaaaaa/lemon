package noveltools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// cleanJSONContent 清理 LLM 返回的 JSON 内容
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

// ValidateNarrationJSON 验证 JSON 格式的解说文案
// 返回解析后的结构化数据和验证结果
func ValidateNarrationJSON(jsonContent string, minLength, maxLength int) (map[string]interface{}, *ValidationResult) {
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

	// 尝试解析 JSON
	var structuredData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &structuredData); err != nil {
		result.IsValid = false
		result.Message = fmt.Sprintf("JSON 解析失败: %v", err)
		return nil, result
	}

	// 验证基本结构
	if structuredData["scenes"] == nil {
		result.IsValid = false
		result.Message = "缺少 scenes 字段"
		return nil, result
	}

	scenes, ok := structuredData["scenes"].([]interface{})
	if !ok {
		result.IsValid = false
		result.Message = "scenes 字段格式错误，应为数组"
		return nil, result
	}

	if len(scenes) < 7 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("分镜数量不足，期望至少7个，实际%d个，但继续生成", len(scenes)))
	}

	// 提取所有解说内容并统计字数
	totalExplanationText := ""
	explanationCount := 0

	for _, sceneInterface := range scenes {
		scene, ok := sceneInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// 分镜级别的解说内容
		if narration, ok := scene["narration"].(string); ok && narration != "" {
			totalExplanationText += narration
			explanationCount++
		}

		// 特写级别的解说内容
		if shots, ok := scene["shots"].([]interface{}); ok {
			for _, shotInterface := range shots {
				shot, ok := shotInterface.(map[string]interface{})
				if !ok {
					continue
				}
				if narration, ok := shot["narration"].(string); ok && narration != "" {
					totalExplanationText += narration
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
	if len(scenes) > 0 {
		firstScene, ok := scenes[0].(map[string]interface{})
		if ok {
			if shots, ok := firstScene["shots"].([]interface{}); ok && len(shots) > 0 {
				firstShot, ok := shots[0].(map[string]interface{})
				if ok {
					if narration, ok := firstShot["narration"].(string); ok {
						charCount := countChineseCharacters(narration)
						result.FirstCloseup = &CloseupValidation{
							Content:   narration,
							CharCount: charCount,
							Valid:     30 <= charCount && charCount <= 32,
							Exists:    true,
						}
					}
				}
			}
			if shots, ok := firstScene["shots"].([]interface{}); ok && len(shots) > 1 {
				secondShot, ok := shots[1].(map[string]interface{})
				if ok {
					if narration, ok := secondShot["narration"].(string); ok {
						charCount := countChineseCharacters(narration)
						result.SecondCloseup = &CloseupValidation{
							Content:   narration,
							CharCount: charCount,
							Valid:     30 <= charCount && charCount <= 32,
							Exists:    true,
						}
					}
				}
			}
		}
	}

	result.Message = "验证通过"
	result.TotalLength = explanationLength
	return structuredData, result
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
