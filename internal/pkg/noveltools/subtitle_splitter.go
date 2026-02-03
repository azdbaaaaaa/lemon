package noveltools

import (
	"regexp"
	"strings"

	"github.com/go-ego/gse"
)

// SubtitleSplitter 字幕文本分割器，用于将文本按自然方式分割为字幕段落
type SubtitleSplitter struct {
	maxLength int            // 每段最大字符数（默认12）
	segmenter *gse.Segmenter // gse 分词器
}

// NewSubtitleSplitter 创建字幕文本分割器实例
func NewSubtitleSplitter(maxLength int) *SubtitleSplitter {
	if maxLength <= 0 {
		maxLength = 12 // 默认值
	}

	// 初始化 gse 分词器
	segmenter, err := gse.New()
	if err != nil {
		// 如果初始化失败，使用空分词器（降级到字符分割）
		segmenter = nil
	}

	return &SubtitleSplitter{
		maxLength: maxLength,
		segmenter: segmenter,
	}
}

// SplitTextNaturally 按句子自然分割文本，确保每句话尽量完整
// 参考 Python 脚本 gen_ass.py 的 split_text_naturally 函数
func (ss *SubtitleSplitter) SplitTextNaturally(text string) []string {
	// 首先按句子分割
	sentenceEndings := []rune{'。', '！', '？', '；', '…', '：'}
	sentences := ss.splitBySentenceEndings(text, sentenceEndings)

	// 如果没有明显的句子分割，按逗号等次级标点分割
	if len(sentences) == 1 && len(cleanSubtitleText(sentences[0])) > ss.maxLength*2 {
		secondaryEndings := []rune{'，', '、', '；'}
		sentences = ss.splitBySentenceEndings(sentences[0], secondaryEndings)
	}

	segments := []string{}
	for _, sentence := range sentences {
		cleanedSentence := cleanSubtitleText(sentence)
		if len(cleanedSentence) <= ss.maxLength {
			segments = append(segments, sentence)
		} else {
			// 句子太长，需要智能分割
			sentenceSegments := ss.splitLongSentenceNaturally(cleanedSentence)
			segments = append(segments, sentenceSegments...)
		}
	}

	// 过滤空段落和单字符段落
	return ss.filterSegments(segments)
}

// splitBySentenceEndings 按句子结束符分割
func (ss *SubtitleSplitter) splitBySentenceEndings(text string, endings []rune) []string {
	sentences := []string{}
	currentSentence := ""

	for _, char := range text {
		currentSentence += string(char)
		if containsRune(endings, char) {
			if strings.TrimSpace(currentSentence) != "" {
				sentences = append(sentences, strings.TrimSpace(currentSentence))
			}
			currentSentence = ""
		}
	}

	if strings.TrimSpace(currentSentence) != "" {
		sentences = append(sentences, strings.TrimSpace(currentSentence))
	}

	return sentences
}

// splitLongSentenceNaturally 智能分割过长的句子
// 使用 gse 分词按词边界分割，避免词组被裁断
func (ss *SubtitleSplitter) splitLongSentenceNaturally(sentence string) []string {
	// 定义自然断开位置的优先级（数字越小优先级越高）
	breakPoints := map[rune]int{
		'，': 1,  // 逗号 - 最自然的断开位置
		'、': 2,  // 顿号
		'；': 3,  // 分号
		'：': 4,  // 冒号
		'的': 5,  // "的"字后面
		'了': 6,  // "了"字后面
		'着': 7,  // "着"字后面
		'过': 8,  // "过"字后面
		'与': 9,  // "与"字后面
		'和': 10, // "和"字后面
		'或': 11, // "或"字后面
		'但': 12, // "但"字前面
		'而': 13, // "而"字前面
		'却': 14, // "却"字前面
		'则': 15, // "则"字前面
	}

	segments := []string{}
	currentSegment := ""

	// 使用 gse 分词获取词汇边界（参考 Python 版本的 jieba.cut）
	var words []string
	if ss.segmenter != nil {
		// 使用 gse 分词
		words = ss.segmenter.Cut(sentence, false)
	} else {
		// 降级：如果没有分词器，按字符分割
		for _, char := range sentence {
			words = append(words, string(char))
		}
	}

	for _, word := range words {
		// 清理词（移除标点符号用于长度计算）
		cleanWord := cleanSubtitleText(word)
		if cleanWord == "" {
			// 如果词只包含标点符号，直接添加到当前段落
			currentSegment += word
			continue
		}

		potentialSegment := currentSegment + word
		cleanPotentialSegment := cleanSubtitleText(potentialSegment)

		if len(cleanPotentialSegment) <= ss.maxLength {
			currentSegment = potentialSegment
		} else {
			// 超出长度限制，需要断开
			if currentSegment != "" {
				// 尝试在当前段落中找到最佳断开位置
				cleanCurrentSegment := cleanSubtitleText(currentSegment)
				bestBreak := ss.findBestBreakPoint(cleanCurrentSegment, breakPoints)
				if bestBreak != nil {
					segments = append(segments, currentSegment)
					currentSegment = word
				} else {
					segments = append(segments, currentSegment)
					currentSegment = word
				}
			} else {
				currentSegment = word
			}

			// 如果单个词过长，强制按字符分割
			cleanCurrentSegment := cleanSubtitleText(currentSegment)
			if len(cleanCurrentSegment) > ss.maxLength {
				charSegments := ss.splitByCharacters(currentSegment)
				segments = append(segments, charSegments[:len(charSegments)-1]...)
				if len(charSegments) > 0 {
					currentSegment = charSegments[len(charSegments)-1]
				} else {
					currentSegment = ""
				}
			}
		}
	}

	if currentSegment != "" {
		segments = append(segments, currentSegment)
	}

	return segments
}

// breakPoint 断开点
type breakPoint struct {
	before string
	after  string
}

// findBestBreakPoint 在文本中找到最佳的自然断开位置
func (ss *SubtitleSplitter) findBestBreakPoint(text string, breakPoints map[rune]int) *breakPoint {
	bestBreak := (*breakPoint)(nil)
	bestPriority := 999

	// 从理想长度位置向前搜索自然断开点
	searchStart := ss.maxLength - 1
	if searchStart >= len(text) {
		searchStart = len(text) - 1
	}
	searchEnd := ss.maxLength / 2
	if searchEnd < 0 {
		searchEnd = 0
	}

	for i := searchStart; i >= searchEnd; i-- {
		if i >= len(text) {
			continue
		}
		char := rune(text[i])

		if priority, ok := breakPoints[char]; ok {
			if priority < bestPriority {
				bestPriority = priority
				// 对于标点符号和"的"、"了"等字，断开位置在字后面
				breakPos := i + 1
				if breakPos > len(text) {
					breakPos = len(text)
				}

				bestBreak = &breakPoint{
					before: strings.TrimSpace(text[:breakPos]),
					after:  strings.TrimSpace(text[breakPos:]),
				}
			}
		}
	}

	return bestBreak
}

// splitByCharacters 按字符强制分割文本
func (ss *SubtitleSplitter) splitByCharacters(text string) []string {
	if len(text) <= ss.maxLength {
		return []string{text}
	}

	segments := []string{}
	start := 0

	for start < len(text) {
		end := start + ss.maxLength
		if end >= len(text) {
			segments = append(segments, text[start:])
			break
		}
		segments = append(segments, text[start:end])
		start = end
	}

	return segments
}

// filterSegments 过滤空段落和单字符段落
func (ss *SubtitleSplitter) filterSegments(segments []string) []string {
	filtered := []string{}

	for i, seg := range segments {
		if strings.TrimSpace(seg) == "" {
			continue
		}

		cleanSeg := cleanSubtitleText(seg)
		if len(cleanSeg) == 1 {
			// 单字符段落，尝试与前一个或后一个段落合并
			if len(filtered) > 0 {
				// 与前一个段落合并
				filtered[len(filtered)-1] += seg
			} else if i+1 < len(segments) && strings.TrimSpace(segments[i+1]) != "" {
				// 与下一个段落合并
				segments[i+1] = seg + segments[i+1]
			} else {
				// 无法合并，保留单字符（避免丢失内容）
				filtered = append(filtered, seg)
			}
		} else {
			filtered = append(filtered, seg)
		}
	}

	return filtered
}

// containsRune 检查rune切片是否包含指定rune
func containsRune(slice []rune, r rune) bool {
	for _, v := range slice {
		if v == r {
			return true
		}
	}
	return false
}

// cleanSubtitleText 清理字幕文本，移除所有标点符号和多余空格
func cleanSubtitleText(text string) string {
	// 移除所有空格
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, "")

	// 移除所有标点符号
	text = regexp.MustCompile(`[，。；：、！？""''（）【】《》〈〉「」『』〔〕\[\]｛｝｜～·…—–,.;:!?"'()\[\]{}|~`+"`"+`@#$%^&*+=<>/\\-]`).ReplaceAllString(text, "")

	return text
}
