package noveltools

import (
	"regexp"
	"strings"
)

// TextCleaner 文本清理器，用于清理TTS文本
type TextCleaner struct{}

// NewTextCleaner 创建文本清理器实例
func NewTextCleaner() *TextCleaner {
	return &TextCleaner{}
}

// CleanTextForTTS 清理文本用于TTS生成，移除括号内的内容和&符号
// 参考 Python 脚本 gen_audio.py 的 clean_text_for_tts 函数
func (tc *TextCleaner) CleanTextForTTS(text string) string {
	// 移除各种括号及其内容
	text = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(text, "")  // 移除圆括号内容
	text = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(text, "") // 移除方括号内容
	text = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(text, "")  // 移除花括号内容
	text = regexp.MustCompile(`（[^）]*）`).ReplaceAllString(text, "")    // 移除中文圆括号内容
	text = regexp.MustCompile(`【[^】]*】`).ReplaceAllString(text, "")    // 移除中文方括号内容

	// 移除&符号
	text = strings.ReplaceAll(text, "&", "")

	// 清理多余的空格
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}
