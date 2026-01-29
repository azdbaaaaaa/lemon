package noveltools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// NarrationValidator 解说内容验证器，用于验证和清理解说内容
type NarrationValidator struct {
	contentFilter *ContentFilter
}

// NewNarrationValidator 创建解说内容验证器实例
func NewNarrationValidator() *NarrationValidator {
	return &NarrationValidator{
		contentFilter: NewContentFilter(),
	}
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid  bool     // 是否有效
	Message  string   // 错误信息或修正后的内容
	Warnings []string // 警告信息列表
	// 分镜1特写验证结果
	FirstCloseup  *CloseupValidation // 第一个特写验证结果
	SecondCloseup *CloseupValidation // 第二个特写验证结果
	TotalLength   int                // 总字数
}

// CloseupValidation 特写验证结果
type CloseupValidation struct {
	Content   string // 解说内容
	CharCount int    // 字数
	Valid     bool   // 是否符合要求（30-32字）
	Exists    bool   // 是否存在
}

// Validate 验证解说内容的质量
//
// Args:
//   - narration: 完整的narration内容
//   - minLength: 解说文本的最小长度（默认1100）
//   - maxLength: 解说文本的最大长度（默认1300）
//   - expectedExplanationCount: 期望的解说内容数量（可选，0表示不验证）
//
// Returns:
//   - ValidationResult: 验证结果
func (nv *NarrationValidator) Validate(
	narration string,
	minLength int,
	maxLength int,
	expectedExplanationCount int,
) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Warnings: make([]string, 0),
	}

	// 检查空内容
	narration = strings.TrimSpace(narration)
	if narration == "" {
		result.IsValid = false
		result.Message = "解说内容为空"
		return result
	}

	// 移除不需要的标签
	cleanedNarration := nv.removeUnwantedTags(narration)

	// 内容审查（仅警告，不阻止）
	if nv.contentFilter != nil {
		checkResult := nv.contentFilter.CheckContent(cleanedNarration)
		if !checkResult.IsSafe {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("检测到敏感内容: %s，但继续生成", strings.Join(checkResult.Issues, "; ")))
		}
	}

	// 提取所有<解说内容>标签内的文本进行字数统计
	explanationPattern := regexp.MustCompile(`<解说内容>(.*?)</解说内容>`)
	explanationMatches := explanationPattern.FindAllStringSubmatch(cleanedNarration, -1)

	if len(explanationMatches) == 0 {
		result.IsValid = false
		result.Message = "未找到解说内容标签"
		return result
	}

	// 计算所有解说内容的总字数
	totalExplanationText := ""
	for _, match := range explanationMatches {
		if len(match) > 1 {
			totalExplanationText += match[1]
		}
	}
	explanationLength := len(strings.TrimSpace(totalExplanationText))

	// 验证特写数量（如果指定了期望值）
	if expectedExplanationCount > 0 && len(explanationMatches) != expectedExplanationCount {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("解说内容数量不正确，期望%d个，实际%d个，但继续生成",
				expectedExplanationCount, len(explanationMatches)))
	}

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

	// 验证分镜1的第一个和第二个特写的字数（30-32字）
	firstCloseup, secondCloseup := nv.findScene1Closeups(cleanedNarration)
	result.FirstCloseup = firstCloseup
	result.SecondCloseup = secondCloseup

	// 自动修复XML标签闭合
	fixedNarration := nv.fixXMLTags(cleanedNarration)

	result.Message = fixedNarration
	result.TotalLength = explanationLength
	return result
}

// ValidateWithAutoFix 验证并自动修复解说内容（如果字数不符合要求，使用LLM改写）
// 参考 Python 脚本 validate_narration.py 的逻辑
func (nv *NarrationValidator) ValidateWithAutoFix(
	ctx context.Context,
	narration string,
	minLength int,
	maxLength int,
	llmProvider LLMProvider,
	maxRetries int,
) (*ValidationResult, error) {
	if maxRetries <= 0 {
		maxRetries = 5
	}

	// 先进行基本验证
	result := nv.Validate(narration, minLength, maxLength, 0)
	updatedContent := result.Message

	// 如果第一个特写不符合要求，尝试改写
	if result.FirstCloseup != nil && result.FirstCloseup.Exists && !result.FirstCloseup.Valid {
		rewritten, err := nv.rewriteCloseupWithLLM(ctx, result.FirstCloseup.Content, llmProvider, maxRetries)
		if err == nil && rewritten != result.FirstCloseup.Content {
			// 替换原内容
			oldTag := fmt.Sprintf("<解说内容>%s</解说内容>", result.FirstCloseup.Content)
			newTag := fmt.Sprintf("<解说内容>%s</解说内容>", rewritten)
			updatedContent = strings.Replace(updatedContent, oldTag, newTag, 1)
			// 更新验证结果
			charCount := nv.countChineseCharacters(rewritten)
			result.FirstCloseup.Content = rewritten
			result.FirstCloseup.CharCount = charCount
			result.FirstCloseup.Valid = 30 <= charCount && charCount <= 32
		}
	}

	// 如果第二个特写不符合要求，尝试改写
	if result.SecondCloseup != nil && result.SecondCloseup.Exists && !result.SecondCloseup.Valid {
		rewritten, err := nv.rewriteCloseupWithLLM(ctx, result.SecondCloseup.Content, llmProvider, maxRetries)
		if err == nil && rewritten != result.SecondCloseup.Content {
			// 替换原内容
			oldTag := fmt.Sprintf("<解说内容>%s</解说内容>", result.SecondCloseup.Content)
			newTag := fmt.Sprintf("<解说内容>%s</解说内容>", rewritten)
			updatedContent = strings.Replace(updatedContent, oldTag, newTag, 1)
			// 更新验证结果
			charCount := nv.countChineseCharacters(rewritten)
			result.SecondCloseup.Content = rewritten
			result.SecondCloseup.CharCount = charCount
			result.SecondCloseup.Valid = 30 <= charCount && charCount <= 32
		}
	}

	// 如果总字数不符合要求，尝试改写所有解说内容
	if result.TotalLength < minLength || result.TotalLength > maxLength {
		rewritten, err := nv.rewriteTotalNarrationWithLLM(ctx, updatedContent, llmProvider, maxRetries)
		if err == nil && rewritten != updatedContent {
			updatedContent = rewritten
			// 重新计算总字数
			explanationPattern := regexp.MustCompile(`<解说内容>(.*?)</解说内容>`)
			explanationMatches := explanationPattern.FindAllStringSubmatch(updatedContent, -1)
			totalText := ""
			for _, match := range explanationMatches {
				if len(match) > 1 {
					totalText += match[1]
				}
			}
			result.TotalLength = len(strings.TrimSpace(totalText))
		}
	}

	result.Message = updatedContent
	return result, nil
}

// findScene1Closeups 查找分镜1的第一个和第二个图片特写的解说内容
func (nv *NarrationValidator) findScene1Closeups(content string) (*CloseupValidation, *CloseupValidation) {
	// 查找分镜1的开始位置
	scene1Match := regexp.MustCompile(`<分镜1>`).FindStringIndex(content)
	if scene1Match == nil {
		return &CloseupValidation{Exists: false}, &CloseupValidation{Exists: false}
	}

	// 从分镜1开始查找
	scene1Start := scene1Match[1]
	scene1Content := content[scene1Start:]

	// 查找分镜1的结束位置（下一个分镜或文件结束）
	endMatch := regexp.MustCompile(`</分镜1>|<分镜2>`).FindStringIndex(scene1Content)
	if endMatch != nil {
		scene1Content = scene1Content[:endMatch[0]]
	}

	// 查找所有图片特写的解说内容
	closeupPattern := regexp.MustCompile(`<图片特写\d+>.*?<解说内容>(.*?)</解说内容>.*?</图片特写\d+>`)
	closeupMatches := closeupPattern.FindAllStringSubmatch(scene1Content, -1)

	var firstCloseup, secondCloseup *CloseupValidation

	if len(closeupMatches) > 0 {
		firstText := strings.TrimSpace(closeupMatches[0][1])
		charCount := nv.countChineseCharacters(firstText)
		firstCloseup = &CloseupValidation{
			Content:   firstText,
			CharCount: charCount,
			Valid:     30 <= charCount && charCount <= 32,
			Exists:    true,
		}
	} else {
		firstCloseup = &CloseupValidation{Exists: false}
	}

	if len(closeupMatches) > 1 {
		secondText := strings.TrimSpace(closeupMatches[1][1])
		charCount := nv.countChineseCharacters(secondText)
		secondCloseup = &CloseupValidation{
			Content:   secondText,
			CharCount: charCount,
			Valid:     30 <= charCount && charCount <= 32,
			Exists:    true,
		}
	} else {
		secondCloseup = &CloseupValidation{Exists: false}
	}

	return firstCloseup, secondCloseup
}

// countChineseCharacters 计算中文字符数量
func (nv *NarrationValidator) countChineseCharacters(text string) int {
	chinesePattern := regexp.MustCompile(`[\u4e00-\u9fff\u3000-\u303f\uff00-\uffef]`)
	matches := chinesePattern.FindAllString(text, -1)
	return len(matches)
}

// rewriteCloseupWithLLM 使用LLM改写特写解说内容，将字数控制在30-32字
func (nv *NarrationValidator) rewriteCloseupWithLLM(
	ctx context.Context,
	originalText string,
	llmProvider LLMProvider,
	maxRetries int,
) (string, error) {
	if llmProvider == nil {
		return originalText, fmt.Errorf("llmProvider is required")
	}

	bestResult := originalText
	bestDistance := 999

	for attempt := 0; attempt < maxRetries; attempt++ {
		var emphasis string
		if attempt == 0 {
			emphasis = "字数必须精准控制在30-32字"
		} else if attempt == 1 {
			emphasis = "字数必须严格控制在30-32字，不能多一字也不能少一字"
		} else {
			emphasis = fmt.Sprintf("字数必须在30-32字之间，当前是第%d次尝试，请务必满足字数要求", attempt+1)
		}

		prompt := fmt.Sprintf(`请将以下解说内容改写，要求：
1. %s（中文字符）
2. 保持原文的核心意思和情感色彩
3. 删除冗余词汇，保留关键情节
4. 语言要流畅自然，适合旁白解说
5. 只返回改写后的内容，不要任何解释

原文：%s

改写后：`, emphasis, originalText)

		rewritten, err := llmProvider.Generate(ctx, prompt)
		if err != nil {
			continue
		}

		rewritten = strings.TrimSpace(rewritten)
		charCount := nv.countChineseCharacters(rewritten)

		// 计算距离目标范围的距离
		var distance int
		if charCount < 30 {
			distance = 30 - charCount
		} else if charCount > 32 {
			distance = charCount - 32
		} else {
			// 在范围内，直接返回
			return rewritten, nil
		}

		// 记录最接近的结果
		if distance < bestDistance {
			bestDistance = distance
			bestResult = rewritten
		}
	}

	// 如果所有重试都失败，返回最接近的结果
	if bestResult != originalText {
		return bestResult, nil
	}
	return originalText, fmt.Errorf("所有%d次重试都未达到30-32字要求", maxRetries)
}

// rewriteTotalNarrationWithLLM 使用LLM重写整个解说内容，将总字数控制在1100-1300字
func (nv *NarrationValidator) rewriteTotalNarrationWithLLM(
	ctx context.Context,
	content string,
	llmProvider LLMProvider,
	maxRetries int,
) (string, error) {
	if llmProvider == nil {
		return content, fmt.Errorf("llmProvider is required")
	}

	// 提取所有解说内容
	explanationPattern := regexp.MustCompile(`<解说内容>(.*?)</解说内容>`)
	explanationMatches := explanationPattern.FindAllStringSubmatch(content, -1)

	if len(explanationMatches) == 0 {
		return content, fmt.Errorf("未找到解说内容")
	}

	allNarrations := make([]string, 0, len(explanationMatches))
	for _, match := range explanationMatches {
		if len(match) > 1 {
			allNarrations = append(allNarrations, strings.TrimSpace(match[1]))
		}
	}

	totalChars := 0
	for _, narration := range allNarrations {
		totalChars += nv.countChineseCharacters(narration)
	}

	// 合并所有解说内容
	combinedText := ""
	for i, narration := range allNarrations {
		combinedText += fmt.Sprintf("解说%d: %s\n\n", i+1, narration)
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		var emphasis string
		if attempt == 0 {
			emphasis = "总字数必须控制在1100-1300字之间"
		} else {
			emphasis = fmt.Sprintf("总字数必须严格控制在1100-1300字之间，当前是第%d次尝试，请务必满足字数要求", attempt+1)
		}

		prompt := fmt.Sprintf(`请重写以下解说内容，要求：
1. %s（中文字符）
2. 保持原文的核心意思和情感色彩
3. 删除冗余词汇，保留关键情节
4. 语言要流畅自然，适合旁白解说
5. 保持解说的数量和顺序不变
6. 每个解说用"解说X: "开头，解说之间用两个换行符分隔
7. 只返回重写后的内容，不要任何解释

原文：
%s

重写后：`, emphasis, combinedText)

		rewritten, err := llmProvider.Generate(ctx, prompt)
		if err != nil {
			continue
		}

		rewritten = strings.TrimSpace(rewritten)

		// 解析重写后的内容
		rewrittenNarrations := make([]string, 0)
		lines := strings.Split(rewritten, "\n\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && strings.Contains(line, "解说") && strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					rewrittenNarrations = append(rewrittenNarrations, strings.TrimSpace(parts[1]))
				}
			}
		}

		// 如果解析失败，尝试直接按换行分割
		if len(rewrittenNarrations) != len(allNarrations) {
			rewrittenNarrations = make([]string, 0)
			for _, line := range strings.Split(rewritten, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					rewrittenNarrations = append(rewrittenNarrations, line)
				}
			}
		}

		// 验证重写后的总字数
		if len(rewrittenNarrations) == len(allNarrations) {
			newTotalChars := 0
			for _, narration := range rewrittenNarrations {
				newTotalChars += nv.countChineseCharacters(narration)
			}

			if 1100 <= newTotalChars && newTotalChars <= 1300 {
				// 替换原内容中的所有解说
				result := content
				for i, original := range allNarrations {
					if i < len(rewrittenNarrations) {
						oldTag := fmt.Sprintf("<解说内容>%s</解说内容>", original)
						newTag := fmt.Sprintf("<解说内容>%s</解说内容>", rewrittenNarrations[i])
						result = strings.Replace(result, oldTag, newTag, 1)
					}
				}
				return result, nil
			}
		}
	}

	return content, fmt.Errorf("所有%d次重试都未达到1100-1300字要求", maxRetries)
}

// removeUnwantedTags 移除不需要的标签
func (nv *NarrationValidator) removeUnwantedTags(content string) string {
	// 定义需要移除的标签列表
	unwantedTags := []string{
		"角色编号", "角色类型", "风格", "文化", "气质",
	}

	cleanedContent := content
	removedTags := make([]string, 0)

	// 移除不需要的标签及其内容
	for _, tag := range unwantedTags {
		// 匹配开始和结束标签之间的内容
		pattern := fmt.Sprintf(`<%s>.*?</%s>`, regexp.QuoteMeta(tag), regexp.QuoteMeta(tag))
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(cleanedContent, -1)
		if len(matches) > 0 {
			for range matches {
				removedTags = append(removedTags, tag)
			}
			cleanedContent = re.ReplaceAllString(cleanedContent, "")
		}

		// 移除单独的开始标签
		singleTagPattern := fmt.Sprintf(`<%s>`, regexp.QuoteMeta(tag))
		if matched, _ := regexp.MatchString(singleTagPattern, cleanedContent); matched {
			removedTags = append(removedTags, tag+"(单标签)")
			re := regexp.MustCompile(singleTagPattern)
			cleanedContent = re.ReplaceAllString(cleanedContent, "")
		}

		// 移除单独的结束标签
		endTagPattern := fmt.Sprintf(`</%s>`, regexp.QuoteMeta(tag))
		if matched, _ := regexp.MatchString(endTagPattern, cleanedContent); matched {
			removedTags = append(removedTags, tag+"(结束标签)")
			re := regexp.MustCompile(endTagPattern)
			cleanedContent = re.ReplaceAllString(cleanedContent, "")
		}
	}

	// 清理多余的空行
	cleanedContent = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(cleanedContent, "\n")

	return strings.TrimSpace(cleanedContent)
}

// fixXMLTags 自动修复XML标签闭合问题
func (nv *NarrationValidator) fixXMLTags(content string) string {
	// 查找所有开始标签
	openTagPattern := regexp.MustCompile(`<([^/\s>]+)[^>]*>`)
	openTagMatches := openTagPattern.FindAllStringSubmatch(content, -1)

	// 查找所有结束标签
	closeTagPattern := regexp.MustCompile(`</([^\s>]+)>`)
	closeTagMatches := closeTagPattern.FindAllStringSubmatch(content, -1)

	// 统计标签出现次数
	openTagCount := make(map[string]int)
	closeTagCount := make(map[string]int)

	for _, match := range openTagMatches {
		if len(match) > 1 {
			tag := match[1]
			openTagCount[tag]++
		}
	}

	for _, match := range closeTagMatches {
		if len(match) > 1 {
			tag := match[1]
			closeTagCount[tag]++
		}
	}

	// 找出未闭合的标签，按出现顺序记录（用于反向添加）
	openTagOrder := make([]string, 0)
	seenTags := make(map[string]bool)
	for _, match := range openTagMatches {
		if len(match) > 1 {
			tag := match[1]
			if !seenTags[tag] {
				openTagOrder = append(openTagOrder, tag)
				seenTags[tag] = true
			}
		}
	}

	// 添加缺失的闭合标签（反向添加，保持嵌套结构）
	fixedContent := content
	for i := len(openTagOrder) - 1; i >= 0; i-- {
		tag := openTagOrder[i]
		openCount := openTagCount[tag]
		closeCount := closeTagCount[tag]
		if openCount > closeCount {
			missingCount := openCount - closeCount
			for j := 0; j < missingCount; j++ {
				fixedContent += fmt.Sprintf("</%s>", tag)
			}
		}
	}

	return fixedContent
}
