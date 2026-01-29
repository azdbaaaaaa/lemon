package noveltools

import (
	"fmt"
	"strings"
)

// ASSGenerator ASS字幕生成器
type ASSGenerator struct{}

// NewASSGenerator 创建ASS字幕生成器实例
func NewASSGenerator() *ASSGenerator {
	return &ASSGenerator{}
}

// GenerateASSContent 生成ASS格式内容
// 参考 Python 脚本 gen_ass.py 的 generate_ass_content 函数
func (ag *ASSGenerator) GenerateASSContent(segmentTimestamps []SegmentTimestamp, title string) string {
	if title == "" {
		title = "Generated Subtitle"
	}

	// ASS文件头部
	assHeader := fmt.Sprintf(`[Script Info]
Title: %s
ScriptType: v4.00+
WrapStyle: 0
ScaledBorderAndShadow: yes
YCbCr Matrix: TV.601
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Microsoft YaHei,36,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,427,1
Style: Highlight,Microsoft YaHei,36,&H0000FFFF,&H000000FF,&H00000000,&H80000000,1,0,0,0,100,100,0,0,1,2,2,2,10,10,427,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`, title)

	// 生成字幕事件
	events := []string{}
	for _, segment := range segmentTimestamps {
		startTime := formatTimeForASS(segment.StartTime)
		endTime := formatTimeForASS(segment.EndTime)
		text := segment.Text

		// 识别关键词并添加高亮效果（简化实现，不依赖jieba）
		keyWord := identifyKeyWord(text)
		highlightedText := text
		if keyWord != "" && strings.Contains(text, keyWord) {
			// 使用ASS标签为关键词添加黄色加粗效果
			replacement := fmt.Sprintf("{\\c&H0000FFFF&\\b1}%s{\\c&H00FFFFFF&\\b0}", keyWord)
			highlightedText = strings.Replace(text, keyWord, replacement, 1)
		}

		// 转义ASS字幕中的特殊字符，特别是汉字双引号
		escapedText := strings.ReplaceAll(highlightedText, "\"", "\\\"")
		escapedText = strings.ReplaceAll(escapedText, "\u201c", "\\\"") // 左双引号
		escapedText = strings.ReplaceAll(escapedText, "\u201d", "\\\"") // 右双引号

		// 生成事件行
		eventLine := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s",
			startTime, endTime, escapedText)
		events = append(events, eventLine)
	}

	return assHeader + strings.Join(events, "\n")
}

// formatTimeForASS 将秒数转换为ASS时间格式 (H:MM:SS.CC)
func formatTimeForASS(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((int(seconds) % 3600) / 60)
	secs := seconds - float64(hours*3600) - float64(minutes*60)
	return fmt.Sprintf("%d:%02d:%05.2f", hours, minutes, secs)
}

// identifyKeyWord 识别文本中的关键词（人名、地名等）
// 简化实现，不依赖jieba，仅做基本识别
func identifyKeyWord(text string) string {
	// 简单的关键词识别：查找常见的专有名词模式
	words := extractPotentialKeywords(text)
	if len(words) > 0 {
		return words[0] // 返回第一个找到的关键词
	}

	return ""
}

// extractPotentialKeywords 提取潜在的关键词
func extractPotentialKeywords(text string) []string {
	keywords := []string{}
	// 简单的模式匹配：查找连续的中文字符（2-4字）
	runes := []rune(text)
	for i := 0; i < len(runes)-1; i++ {
		if isChinese(runes[i]) && isChinese(runes[i+1]) {
			// 找到2个连续的中文字符
			word := string(runes[i : i+2])
			if i+2 < len(runes) && isChinese(runes[i+2]) {
				word = string(runes[i : i+3])
				if i+3 < len(runes) && isChinese(runes[i+3]) {
					word = string(runes[i : i+4])
				}
			}
			if len(word) >= 2 && len(word) <= 4 {
				keywords = append(keywords, word)
				i += len([]rune(word)) - 1 // 跳过已匹配的字符
			}
		}
	}
	return keywords
}

// isChinese 检查rune是否为中文字符
func isChinese(r rune) bool {
	return r >= 0x4e00 && r <= 0x9fff
}
