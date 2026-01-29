package noveltools

import (
	"regexp"
	"strings"
)

// NarrationExtractor 解说内容提取器
type NarrationExtractor struct{}

// NewNarrationExtractor 创建解说内容提取器实例
func NewNarrationExtractor() *NarrationExtractor {
	return &NarrationExtractor{}
}

// ExtractNarrationContent 从narration文本中提取所有解说内容
// 参考 Python 脚本 gen_audio.py 的 extract_narration_content 函数
func (ne *NarrationExtractor) ExtractNarrationContent(narrationText string) []string {
	narrationContents := []string{}

	// 使用正则表达式提取所有<解说内容>标签中的内容
	// 注意：文件中的解说内容标签可能没有结束标签，内容直接跟在开始标签后面直到下一个标签
	pattern := regexp.MustCompile(`<解说内容>([^<]+)`)
	matches := pattern.FindAllStringSubmatch(narrationText, -1)

	for _, match := range matches {
		if len(match) > 1 {
			// 清理文本，移除多余的空白字符
			cleanNarration := strings.TrimSpace(match[1])
			if cleanNarration != "" {
				narrationContents = append(narrationContents, cleanNarration)
			}
		}
	}

	return narrationContents
}
