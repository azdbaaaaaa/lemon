package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/storage"
	novelrepo "lemon/internal/repository/novel"
	resourceRepo "lemon/internal/repository/resource"
)

// NovelService 小说服务
// 用途：对单一章节生成解说，并落库到 chapters.narration_text
type NovelService struct {
	resourceRepo *resourceRepo.ResourceRepo
	novelRepo    novelrepo.NovelRepository
	chapterRepo  novelrepo.ChapterRepository
	storage      storage.Storage
	llmProvider  noveltools.LLMProvider
}

// NewNovelService 创建小说服务
func NewNovelService(
	resourceRepo *resourceRepo.ResourceRepo,
	novelRepo novelrepo.NovelRepository,
	chapterRepo novelrepo.ChapterRepository,
	storage storage.Storage,
	llmProvider noveltools.LLMProvider,
) *NovelService {
	return &NovelService{
		resourceRepo: resourceRepo,
		novelRepo:    novelRepo,
		chapterRepo:  chapterRepo,
		storage:      storage,
		llmProvider:  llmProvider,
	}
}

// GenerateNarrationForChapter 为单一章节生成解说文本，并更新到章节的 narration_text 字段
func (s *NovelService) GenerateNarrationForChapter(ctx context.Context, chapterID string) (string, error) {
	if s.chapterRepo == nil {
		return "", fmt.Errorf("chapterRepo is required")
	}
	if s.llmProvider == nil {
		return "", fmt.Errorf("llmProvider is required")
	}
	chapterID = strings.TrimSpace(chapterID)
	if chapterID == "" {
		return "", fmt.Errorf("chapterID is required")
	}

	ch, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", err
	}

	totalChapters, err := s.getTotalChapters(ctx, ch.NovelID)
	if err != nil {
		return "", err
	}

	generator := noveltools.NewNarrationGenerator(s.llmProvider)
	narrationText, err := generator.Generate(ctx, ch.ChapterText, ch.Sequence, totalChapters)
	if err != nil {
		return "", err
	}

	narrationText = strings.TrimSpace(narrationText)
	if narrationText == "" {
		return "", fmt.Errorf("generated narrationText is empty")
	}

	validatedText, err := s.validateAndFixNarration(ctx, narrationText, false)
	if err != nil {
		return "", err
	}

	if err := s.chapterRepo.UpdateNarrationText(ctx, ch.ID, validatedText); err != nil {
		return "", err
	}
	return validatedText, nil
}

func (s *NovelService) getTotalChapters(ctx context.Context, novelID string) (int, error) {
	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return 0, err
	}
	if len(chapters) == 0 {
		return 0, fmt.Errorf("no chapters found for novelID=%s", novelID)
	}
	return len(chapters), nil
}

// CreateNovelFromResource 第一步：根据资源ID获取小说内容，然后创建小说
// 返回创建的小说ID
func (s *NovelService) CreateNovelFromResource(ctx context.Context, resourceID, userID, workflowID string) (string, error) {
	if s.resourceRepo == nil {
		return "", fmt.Errorf("resourceRepo is required")
	}
	if s.novelRepo == nil {
		return "", fmt.Errorf("novelRepo is required")
	}

	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "", fmt.Errorf("resourceID is required")
	}

	_, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		return "", fmt.Errorf("failed to find resource: %w", err)
	}

	novelID := id.New()
	novelEntity := &novel.Novel{
		ID:         novelID,
		ResourceID: resourceID,
		UserID:     userID,
		WorkflowID: workflowID,
	}

	if err := s.novelRepo.Create(ctx, novelEntity); err != nil {
		return "", fmt.Errorf("failed to create novel: %w", err)
	}

	return novelID, nil
}

// SplitNovelIntoChapters 第二步：根据小说内容切分章节，然后插入章节数据
// 需要先从资源中读取内容，然后切分并保存章节
func (s *NovelService) SplitNovelIntoChapters(ctx context.Context, novelID string, targetChapters int) error {
	if s.novelRepo == nil {
		return fmt.Errorf("novelRepo is required")
	}
	if s.resourceRepo == nil {
		return fmt.Errorf("resourceRepo is required")
	}
	if s.storage == nil {
		return fmt.Errorf("storage is required")
	}
	if s.chapterRepo == nil {
		return fmt.Errorf("chapterRepo is required")
	}

	novelID = strings.TrimSpace(novelID)
	if novelID == "" {
		return fmt.Errorf("novelID is required")
	}

	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("failed to find novel: %w", err)
	}

	res, err := s.resourceRepo.FindByID(ctx, novelEntity.ResourceID)
	if err != nil {
		return fmt.Errorf("failed to find resource: %w", err)
	}

	reader, err := s.storage.Download(ctx, res.StorageKey)
	if err != nil {
		return fmt.Errorf("failed to download resource: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read resource content: %w", err)
	}

	splitter := noveltools.NewChapterSplitter()
	segments := splitter.Split(string(content), targetChapters)
	if len(segments) == 0 {
		return fmt.Errorf("no chapters split from novel content")
	}

	for i, seg := range segments {
		chapterID := id.New()
		chapterEntity := &novel.Chapter{
			ID:          chapterID,
			NovelID:     novelID,
			WorkflowID:  novelEntity.WorkflowID,
			UserID:      novelEntity.UserID,
			Sequence:    i + 1,
			Title:       seg.Title,
			ChapterText: seg.Text,
		}

		if err := s.chapterRepo.Create(ctx, chapterEntity); err != nil {
			return fmt.Errorf("failed to create chapter %d: %w", i+1, err)
		}
	}

	return nil
}

// GenerateNarrationsForAllChapters 第三步：并发地根据每一章节内容生成章节对应的解说文案
func (s *NovelService) GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error {
	if s.chapterRepo == nil {
		return fmt.Errorf("chapterRepo is required")
	}
	if s.llmProvider == nil {
		return fmt.Errorf("llmProvider is required")
	}

	novelID = strings.TrimSpace(novelID)
	if novelID == "" {
		return fmt.Errorf("novelID is required")
	}

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
			narrationText, err := generator.Generate(ctx, chapter.ChapterText, chapter.Sequence, totalChapters)
			if err != nil {
				errCh <- fmt.Errorf("failed to generate narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			narrationText = strings.TrimSpace(narrationText)
			if narrationText == "" {
				errCh <- fmt.Errorf("generated narrationText is empty for chapter %d", chapter.Sequence)
				return
			}

			validatedText, err := s.validateAndFixNarration(ctx, narrationText, false)
			if err != nil {
				errCh <- fmt.Errorf("failed to validate narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			if err := s.chapterRepo.UpdateNarrationText(ctx, chapter.ID, validatedText); err != nil {
				errCh <- fmt.Errorf("failed to update narration for chapter %d: %w", chapter.Sequence, err)
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

// validateAndFixNarration 验证并修复解说文案
// 包括：长度检查、XML标签修复、移除不需要的标签、内容审查
// 如果启用自动改写，会使用LLM改写不符合字数要求的解说
func (s *NovelService) validateAndFixNarration(ctx context.Context, narrationText string, autoFix bool) (string, error) {
	validator := noveltools.NewNarrationValidator()

	var result *noveltools.ValidationResult
	var err error

	if autoFix && s.llmProvider != nil {
		// 使用自动改写功能（参考 Python 脚本 validate_narration.py --auto-fix）
		result, err = validator.ValidateWithAutoFix(ctx, narrationText, 1100, 1300, s.llmProvider, 5)
		if err != nil {
			// 自动改写失败，降级到基本验证
			result = validator.Validate(narrationText, 1100, 1300, 0)
		}
	} else {
		// 基本验证（不自动改写）
		result = validator.Validate(narrationText, 1100, 1300, 0)
	}

	if !result.IsValid {
		return "", fmt.Errorf("narration validation failed: %s", result.Message)
	}

	// 检查分镜1特写的验证结果（仅警告，不阻止）
	if result.FirstCloseup != nil && result.FirstCloseup.Exists && !result.FirstCloseup.Valid {
		// 可以记录警告日志
		_ = result.FirstCloseup
	}
	if result.SecondCloseup != nil && result.SecondCloseup.Exists && !result.SecondCloseup.Valid {
		// 可以记录警告日志
		_ = result.SecondCloseup
	}

	return result.Message, nil
}
