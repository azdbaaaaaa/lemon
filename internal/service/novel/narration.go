package novel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
)

// GenerateNarrationForChapter 为单一章节生成解说文本，并保存到 narrations 表
// 返回的是 JSON 格式的字符串，实际存储的是结构化数据
func (s *novelService) GenerateNarrationForChapter(ctx context.Context, chapterID string) (string, error) {
	ch, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", err
	}

	totalChapters, err := s.getTotalChapters(ctx, ch.NovelID)
	if err != nil {
		return "", err
	}

	generator := noveltools.NewNarrationGenerator(s.llmProvider)
	prompt, narrationText, err := generator.GenerateWithPrompt(ctx, ch.ChapterText, ch.Sequence, totalChapters)
	if err != nil {
		return "", err
	}

	narrationText = strings.TrimSpace(narrationText)
	if narrationText == "" {
		return "", fmt.Errorf("generated narrationText is empty")
	}

	// 步骤1: 内容审查和过滤（参考 Python 的 audit_and_filter_narration）
	// 极度宽松模式：仅提示，不阻断
	filteredNarration, err := s.auditAndFilterNarration(ctx, narrationText, ch.Sequence)
	if err != nil {
		// 即使审查出错，也继续使用原始内容（极度宽松模式）
		filteredNarration = narrationText
	}

	// 步骤2: 验证 JSON 格式并解析为结构化数据
	structuredContent, validationResult := noveltools.ValidateNarrationJSON(filteredNarration, 1100, 1300)
	if !validationResult.IsValid {
		return "", fmt.Errorf("narration validation failed: %s", validationResult.Message)
	}

	// 保存到 narrations 表
	narrationID := id.New()
	narrationEntity := &novel.Narration{
		ID:        narrationID,
		ChapterID: ch.ID,
		UserID:    ch.UserID,
		Content:   structuredContent,
		Prompt:    prompt,
		Status:    "completed",
	}

	// 如果已存在解说文案，先软删除旧的
	existingNarration, err := s.narrationRepo.FindByChapterID(ctx, ch.ID)
	if err == nil {
		_ = s.narrationRepo.Delete(ctx, existingNarration.ID)
	}

	if err := s.narrationRepo.Create(ctx, narrationEntity); err != nil {
		return "", fmt.Errorf("failed to save narration: %w", err)
	}

	// 返回 JSON 字符串（LLM 生成的原始 JSON 内容）
	return filteredNarration, nil
}

// GenerateNarrationsForAllChapters 第三步：并发地根据每一章节内容生成章节对应的解说文案
func (s *novelService) GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error {
	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("failed to find chapters: %w", err)
	}
	if len(chapters) == 0 {
		return fmt.Errorf("no chapters found for novelID=%s", novelID)
	}

	totalChapters := len(chapters)
	var wg sync.WaitGroup
	errCh := make(chan error, totalChapters)

	for _, ch := range chapters {
		wg.Add(1)
		go func(chapter *novel.Chapter) {
			defer wg.Done()

			generator := noveltools.NewNarrationGenerator(s.llmProvider)
			prompt, narrationText, err := generator.GenerateWithPrompt(ctx, chapter.ChapterText, chapter.Sequence, totalChapters)
			if err != nil {
				errCh <- fmt.Errorf("failed to generate narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			narrationText = strings.TrimSpace(narrationText)
			if narrationText == "" {
				errCh <- fmt.Errorf("generated narrationText is empty for chapter %d", chapter.Sequence)
				return
			}

			// 步骤1: 内容审查和过滤（参考 Python 的 audit_and_filter_narration）
			// 极度宽松模式：仅提示，不阻断
			filteredNarration, err := s.auditAndFilterNarration(ctx, narrationText, chapter.Sequence)
			if err != nil {
				// 即使审查出错，也继续使用原始内容（极度宽松模式）
				filteredNarration = narrationText
			}

			// 步骤2: 验证 JSON 格式并解析为结构化数据
			structuredContent, validationResult := noveltools.ValidateNarrationJSON(filteredNarration, 1100, 1300)
			if !validationResult.IsValid {
				errCh <- fmt.Errorf("failed to validate narration for chapter %d: %s", chapter.Sequence, validationResult.Message)
				return
			}

			// 保存到 narrations 表
			narrationID := id.New()
			narrationEntity := &novel.Narration{
				ID:        narrationID,
				ChapterID: chapter.ID,
				UserID:    chapter.UserID,
				Content:   structuredContent,
				Prompt:    prompt,
				Status:    "completed",
			}

			// 如果已存在解说文案，先软删除旧的
			existingNarration, err := s.narrationRepo.FindByChapterID(ctx, chapter.ID)
			if err == nil {
				_ = s.narrationRepo.Delete(ctx, existingNarration.ID)
			}

			if err := s.narrationRepo.Create(ctx, narrationEntity); err != nil {
				errCh <- fmt.Errorf("failed to save narration for chapter %d: %w", chapter.Sequence, err)
				return
			}
		}(ch)
	}

	wg.Wait()
	close(errCh)

	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to generate narrations for %d chapters: %v", len(errors), errors)
	}

	return nil
}

// GetNarration 根据章节ID获取解说文案
func (s *novelService) GetNarration(ctx context.Context, chapterID string) (*novel.Narration, error) {
	return s.narrationRepo.FindByChapterID(ctx, chapterID)
}

// getTotalChapters 获取小说的总章节数
func (s *novelService) getTotalChapters(ctx context.Context, novelID string) (int, error) {
	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return 0, err
	}
	if len(chapters) == 0 {
		return 0, fmt.Errorf("no chapters found for novelID=%s", novelID)
	}
	return len(chapters), nil
}

// auditAndFilterNarration 对生成的解说内容进行审查和过滤（极度宽松模式）
// 参考 Python 的 audit_and_filter_narration 方法
// 仅提示，不阻断，即使检测到敏感内容也返回原始内容
func (s *novelService) auditAndFilterNarration(ctx context.Context, narration string, chapterNum int) (string, error) {
	contentFilter := noveltools.NewContentFilter()

	// 检查内容是否包含违禁词汇（仅提示，不阻断）
	checkResult := contentFilter.CheckContent(narration)

	if !checkResult.IsSafe {
		// 记录警告日志（在实际环境中可以使用 log 包）
		// log.Warn().Int("chapter_num", chapterNum).Strs("issues", checkResult.Issues).
		// 	Msg("检测到敏感内容，但继续生成")
		_ = checkResult.Issues // 暂时忽略，避免未使用变量警告
	}

	// 无论是否检测到敏感内容，都返回原始内容（极度宽松模式）
	return narration, nil
}
