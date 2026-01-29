package noveltools

import (
	"strings"
)

// ContentFilter 内容过滤器，用于检测和替换违禁词汇和敏感内容
type ContentFilter struct {
	// 违禁词汇列表（极度宽松 - 仅保留最极端的敏感内容）
	forbiddenWords map[string]bool

	// 敏感词替换映射
	wordReplacements map[string]string

	// 严重违禁词汇（直接移除）
	seriousForbiddenWords map[string]bool

	// 词汇替换映射（用于替换敏感但可接受的词汇）
	wordSubstitutions map[string]string
}

// NewContentFilter 创建内容过滤器实例
func NewContentFilter() *ContentFilter {
	return &ContentFilter{
		forbiddenWords: map[string]bool{
			"毒品": true,
			"强暴": true,
		},
		wordReplacements: map[string]string{
			"罪犯":   "嫌疑人",
			"通缉犯":  "TJ",
			"警察":   "jc",
			"监狱":   "牢狱",
			"遗体":   "YT",
			"死":    "S",
			"上吊":   "SD",
			"自杀":   "ZS",
			"跳楼":   "TL",
			"尸体":   "ST",
			"回房睡觉": "回房休息",
			"睡觉":   "休息",
		},
		seriousForbiddenWords: map[string]bool{
			"双修":  true,
			"采补":  true,
			"吸精":  true,
			"吸精气": true,
			"乱摸":  true,
			"乱动":  true,
			"赤裸裸": true,
			"服侍":  true,
			"爆浆":  true,
			"床上":  true,
			"大宝贝": true,
			"勾引":  true,
			"色情":  true,
			"偷人":  true,
			"鼎炉":  true,
			"春药":  true,
			"媚药":  true,
			"软床":  true,
			"丝袜":  true,
			"催情":  true,
			"允吸":  true,
			"毒品":  true,
			"上床":  true,
			"强暴":  true,
			"性欲":  true,
		},
		wordSubstitutions: map[string]string{
			"拥抱": "相伴",
			"温柔": "和善",
			"温热": "温暖",
			"目光": "视线",
			"欲望": "愿望",
			"互动": "交流",
			"诱惑": "吸引",
			"怀里": "身边",
			"大腿": "腿部",
			"抱起": "扶起",
			"姿势": "动作",
		},
	}
}

// CheckResult 检查结果
type CheckResult struct {
	IsSafe bool     // 是否通过检查
	Issues []string // 发现的问题列表
}

// CheckContent 检查内容是否包含违禁词汇或敏感内容
//
// Args:
//
//	content: 要检查的内容
//
// Returns:
//
//	CheckResult: 检查结果，包含是否通过和问题列表
func (cf *ContentFilter) CheckContent(content string) *CheckResult {
	result := &CheckResult{
		IsSafe: true,
		Issues: make([]string, 0),
	}

	// 检查违禁词汇
	for word := range cf.forbiddenWords {
		if strings.Contains(content, word) {
			result.IsSafe = false
			result.Issues = append(result.Issues, "发现违禁词汇: "+word)
		}
	}

	// 检查严重违禁词汇
	for word := range cf.seriousForbiddenWords {
		if strings.Contains(content, word) {
			result.IsSafe = false
			result.Issues = append(result.Issues, "发现严重违禁词汇: "+word)
		}
	}

	return result
}

// FilterContent 过滤内容，替换敏感词汇
//
// Args:
//
//	content: 原始内容
//
// Returns:
//
//	string: 过滤后的内容
func (cf *ContentFilter) FilterContent(content string) string {
	filteredContent := content

	// 执行词汇替换
	for original, replacement := range cf.wordReplacements {
		filteredContent = strings.ReplaceAll(filteredContent, original, replacement)
	}

	// 执行词汇替换（用于替换敏感但可接受的词汇）
	for word, substitute := range cf.wordSubstitutions {
		filteredContent = strings.ReplaceAll(filteredContent, word, substitute)
	}

	// 移除严重违禁词汇
	for word := range cf.seriousForbiddenWords {
		filteredContent = strings.ReplaceAll(filteredContent, word, "")
	}

	// 清理多余空格
	filteredContent = cleanWhitespace(filteredContent)

	return filteredContent
}

// cleanWhitespace 清理多余的空格和换行
func cleanWhitespace(content string) string {
	// 将多个连续空格替换为单个空格
	content = strings.ReplaceAll(content, "  ", " ")
	// 将多个连续换行替换为单个换行
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(content)
}

// ProcessContent 处理内容：检查并过滤
// 这是一个便捷方法，同时执行检查和过滤
//
// Args:
//
//	content: 原始内容
//
// Returns:
//
//	filteredContent: 过滤后的内容
//	checkResult: 检查结果
func (cf *ContentFilter) ProcessContent(content string) (filteredContent string, checkResult *CheckResult) {
	checkResult = cf.CheckContent(content)
	filteredContent = cf.FilterContent(content)
	return filteredContent, checkResult
}
