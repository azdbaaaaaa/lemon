package noveltools

import (
	"bufio"
	"regexp"
	"strings"
)

// ChapterSegment 表示按章节切分后的一段内容
type ChapterSegment struct {
	Title string // 章节标题（如无法识别标题则为内容前若干字）
	Text  string // 章节全文
}

// ChapterSplitter 章节切分器，用于将小说内容切分为若干章节
type ChapterSplitter struct {
	// 默认目标章节数（当 targetChapters <= 0 时使用）
	defaultTargetChapters int
	// 是否合并章节（当章节数超过目标章节数时）
	// true: 合并章节（将多个章节合并为一个）
	// false: 只保留前N章（默认）
	mergeWhenTooMany bool
	// 最小章节长度（字符数），小于此长度的章节会被过滤
	// 0 表示只要不为空就保留（默认）
	minChapterLength int
}

// NewChapterSplitter 创建章节切分器实例
func NewChapterSplitter() *ChapterSplitter {
	return &ChapterSplitter{
		defaultTargetChapters: 50,
		mergeWhenTooMany:      false, // 默认不合并，只保留前N章
		minChapterLength:      0,     // 默认只要不为空就保留
	}
}

// SetMergeWhenTooMany 设置是否在章节数过多时合并章节
func (cs *ChapterSplitter) SetMergeWhenTooMany(merge bool) {
	cs.mergeWhenTooMany = merge
}

// SetMinChapterLength 设置最小章节长度（字符数）
// 小于此长度的章节会被过滤
// 0 表示只要不为空就保留（默认）
func (cs *ChapterSplitter) SetMinChapterLength(length int) {
	cs.minChapterLength = length
}

// Split 将小说内容切分为若干章节
//
// 逻辑：
//  1. 先按常见章节标题模式切分（第X章 / Chapter N / 章节 N）
//  2. 若识别到章节标题，保持一章一章切分，如果章节数超过 targetChapters，只保留前 targetChapters 章
//  3. 若无法识别章节标题，则按长度平均切分为 targetChapters 段
//
// Args:
//   - novelContent: 小说原始内容
//   - targetChapters: 目标章节数（<= 0 时使用默认值）
//
// Returns:
//   - []ChapterSegment: 切分后的章节列表
func (cs *ChapterSplitter) Split(novelContent string, targetChapters int) []ChapterSegment {
	novelContent = normalizeNovelText(novelContent)
	if novelContent == "" {
		return nil
	}
	if targetChapters <= 0 {
		targetChapters = cs.defaultTargetChapters
	}

	if chunks := splitByChapterTitles(novelContent, cs.minChapterLength); len(chunks) >= 2 {
		// 如果识别到章节标题，保持一章一章切分
		// 如果章节数超过 targetChapters，根据配置决定是合并还是截取
		if len(chunks) > targetChapters {
			if cs.mergeWhenTooMany {
				// 合并章节
				chunks = mergeIfTooMany(chunks, targetChapters)
			} else {
				// 只保留前 targetChapters 章（默认行为）
				chunks = chunks[:targetChapters]
			}
		}
		return wrapSegments(chunks)
	}

	// 无法识别章节标题时，按长度平均切分
	chunks := splitByLength(novelContent, targetChapters)
	return wrapSegments(chunks)
}

// ----- 内部实现，与原 processor 中算法基本一致，仅去掉资源/存储依赖 -----

func normalizeNovelText(s string) string {
	// 对齐 python 逻辑：re.sub(r'\n\s*\n', '\n\n', novel_content).strip()
	lines := strings.Split(s, "\n")
	var out []string
	blank := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !blank {
				out = append(out, "")
				blank = true
			}
			continue
		}
		blank = false
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

var chapterTitlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^第[一二三四五六七八九十百千万0-9\d]+章[^\n]*`),
	regexp.MustCompile(`(?im)^chapter\s*\d+[^\n]*`),
	regexp.MustCompile(`(?im)^章节\s*\d+[^\n]*`),
}

func splitByChapterTitles(novelContent string, minLength int) []string {
	var matches []int
	for _, re := range chapterTitlePatterns {
		idxs := re.FindAllStringIndex(novelContent, -1)
		if len(idxs) >= 2 {
			for _, idx := range idxs {
				matches = append(matches, idx[0])
			}
			break
		}
	}
	if len(matches) < 2 {
		return nil
	}

	matches = uniqueSortedInts(matches)
	var chapters []string
	for i := 0; i < len(matches); i++ {
		start := matches[i]
		end := len(novelContent)
		if i+1 < len(matches) {
			end = matches[i+1]
		}
		ch := strings.TrimSpace(novelContent[start:end])
		// 根据配置的最小长度检查章节
		if ch != "" {
			if minLength > 0 {
				// 如果设置了最小长度，检查章节长度
				if len([]rune(ch)) >= minLength {
					chapters = append(chapters, ch)
				}
			} else {
				// 默认：只要不为空就保留
				chapters = append(chapters, ch)
			}
		}
	}
	return chapters
}

func splitByLength(novelContent string, targetChapters int) []string {
	r := []rune(novelContent)
	total := len(r)
	if total == 0 {
		return nil
	}
	chunk := total / targetChapters
	if chunk <= 0 {
		return []string{novelContent}
	}

	chapters := make([]string, 0, targetChapters)
	for i := 0; i < targetChapters; i++ {
		start := i * chunk
		end := (i + 1) * chunk
		if i == targetChapters-1 || end > total {
			end = total
		}
		part := strings.TrimSpace(string(r[start:end]))
		if part != "" {
			chapters = append(chapters, part)
		}
	}
	return chapters
}

// mergeIfTooMany 合并章节（当章节数超过目标章节数时）
// 将多个章节按长度合并，使最终章节数接近目标章节数
// 注意：此函数只在 ChapterSplitter.mergeWhenTooMany = true 时使用
func mergeIfTooMany(chapters []string, targetChapters int) []string {
	if targetChapters <= 0 || len(chapters) <= targetChapters {
		return chapters
	}
	totalLen := 0
	for _, ch := range chapters {
		totalLen += len([]rune(ch))
	}
	targetLen := totalLen / targetChapters

	merged := make([]string, 0, targetChapters)
	var cur strings.Builder
	curLen := 0

	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			merged = append(merged, s)
		}
		cur.Reset()
		curLen = 0
	}

	for _, ch := range chapters {
		chLen := len([]rune(ch))
		if curLen < targetLen && curLen > 0 {
			cur.WriteString("\n\n")
			curLen += 2
		}
		cur.WriteString(ch)
		curLen += chLen
		if curLen >= targetLen && len(merged) < targetChapters-1 {
			flush()
		}
	}
	flush()

	if len(merged) > targetChapters {
		return merged[:targetChapters]
	}
	return merged
}

func uniqueSortedInts(a []int) []int {
	if len(a) == 0 {
		return a
	}
	m := make(map[int]struct{}, len(a))
	for _, v := range a {
		m[v] = struct{}{}
	}
	out := make([]int, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func extractChapterTitle(text string) string {
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		for _, re := range chapterTitlePatterns {
			if re.MatchString(line) {
				return line
			}
		}
		if len([]rune(line)) > 30 {
			return string([]rune(line)[:30])
		}
		return line
	}
	return ""
}

func wrapSegments(chunks []string) []ChapterSegment {
	segments := make([]ChapterSegment, 0, len(chunks))
	for _, ch := range chunks {
		title := extractChapterTitle(ch)
		segments = append(segments, ChapterSegment{
			Title: title,
			Text:  ch,
		})
	}
	return segments
}
