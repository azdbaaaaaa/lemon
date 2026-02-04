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

// NarrationService 章节解说服务接口
// 定义章节解说相关的能力
type NarrationService interface {
	// GenerateNarrationForChapter 为单一章节生成解说文本
	GenerateNarrationForChapter(ctx context.Context, chapterID string) (string, error)

	// GenerateNarrationsForAllChapters 并发地为所有章节生成解说文本
	GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error

	// GetNarration 根据章节ID获取章节解说（返回最新版本）
	GetNarration(ctx context.Context, chapterID string) (*novel.Narration, error)

	// GetNarrationByVersion 根据章节ID和版本号获取章节解说
	GetNarrationByVersion(ctx context.Context, chapterID string, version int) (*novel.Narration, error)

	// SetNarrationVersion 设置章节解说的版本号
	SetNarrationVersion(ctx context.Context, narrationID string, version int) error

	// GetNarrationVersions 获取章节的所有版本号
	GetNarrationVersions(ctx context.Context, chapterID string) ([]int, error)
}

// GenerateNarrationForChapter 为单一章节生成章节解说，并保存到 chapter_narrations 表
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
	// 传递章节字数，用于根据章节长度调整 prompt 要求
	prompt, narrationText, err := generator.GenerateWithPrompt(ctx, ch.ChapterText, ch.Sequence, totalChapters, ch.WordCount)
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

	// 步骤2: 解析 JSON 格式并验证
	jsonContent, err := noveltools.ParseNarrationJSON(filteredNarration)
	if err != nil {
		return "", fmt.Errorf("narration parsing failed: %w", err)
	}

	// 基本验证：至少要有场景
	if len(jsonContent.Scenes) == 0 {
		return "", fmt.Errorf("narration validation failed: 缺少 scenes 字段或 scenes 为空")
	}

	// 生成下一个版本号（自动递增）
	nextVersion, err := s.getNextNarrationVersion(ctx, ch.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get next version: %w", err)
	}

	// 保存到 narrations 表（不删除旧版本，支持多版本并存）
	narrationID := id.New()
	narrationEntity := &novel.Narration{
		ID:        narrationID,
		ChapterID: ch.ID,
		UserID:    ch.UserID,
		Prompt:    prompt,
		Version:   nextVersion,
		Status:    "completed",
	}

	if err := s.narrationRepo.Create(ctx, narrationEntity); err != nil {
		return "", fmt.Errorf("failed to save narration: %w", err)
	}

	// 步骤3: 将场景和镜头转换为实体并保存到独立的表中
	scenes, shots, err := noveltools.ConvertToScenesAndShots(narrationID, ch.ID, ch.UserID, nextVersion, jsonContent)
	if err != nil {
		return "", fmt.Errorf("failed to convert scenes and shots: %w", err)
	}

	// 批量保存场景
	if len(scenes) > 0 {
		if err := s.sceneRepo.CreateMany(ctx, scenes); err != nil {
			return "", fmt.Errorf("failed to save scenes: %w", err)
		}
	}

	// 批量保存镜头
	if len(shots) > 0 {
		if err := s.shotRepo.CreateMany(ctx, shots); err != nil {
			return "", fmt.Errorf("failed to save shots: %w", err)
		}
	}

	// 返回 JSON 字符串（LLM 生成的原始 JSON 内容）
	return filteredNarration, nil
}

// GenerateNarrationsForAllChapters 第三步：并发地根据每一章节内容生成章节对应的章节解说
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
			// 传递章节字数，用于根据章节长度调整 prompt 要求
			prompt, narrationText, err := generator.GenerateWithPrompt(ctx, chapter.ChapterText, chapter.Sequence, totalChapters, chapter.WordCount)
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

			// 步骤2: 解析 JSON 格式并验证
			jsonContent, err := noveltools.ParseNarrationJSON(filteredNarration)
			if err != nil {
				errCh <- fmt.Errorf("failed to parse narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 基本验证：至少要有场景
			if len(jsonContent.Scenes) == 0 {
				errCh <- fmt.Errorf("failed to validate narration for chapter %d: 缺少 scenes 字段或 scenes 为空", chapter.Sequence)
				return
			}

			// 生成下一个版本号（自动递增）
			nextVersion, err := s.getNextNarrationVersion(ctx, chapter.ID)
			if err != nil {
				errCh <- fmt.Errorf("failed to get next version for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 保存到 narrations 表（不删除旧版本，支持多版本并存）
			narrationID := id.New()
			narrationEntity := &novel.Narration{
				ID:        narrationID,
				ChapterID: chapter.ID,
				UserID:    chapter.UserID,
				Prompt:    prompt,
				Version:   nextVersion,
				Status:    "completed",
			}

			if err := s.narrationRepo.Create(ctx, narrationEntity); err != nil {
				errCh <- fmt.Errorf("failed to save narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 步骤3: 将场景和镜头转换为实体并保存到独立的表中
			scenes, shots, err := noveltools.ConvertToScenesAndShots(narrationID, chapter.ID, chapter.UserID, nextVersion, jsonContent)
			if err != nil {
				errCh <- fmt.Errorf("failed to convert scenes and shots for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 批量保存场景
			if len(scenes) > 0 {
				if err := s.sceneRepo.CreateMany(ctx, scenes); err != nil {
					errCh <- fmt.Errorf("failed to save scenes for chapter %d: %w", chapter.Sequence, err)
					return
				}
			}

			// 批量保存镜头
			if len(shots) > 0 {
				if err := s.shotRepo.CreateMany(ctx, shots); err != nil {
					errCh <- fmt.Errorf("failed to save shots for chapter %d: %w", chapter.Sequence, err)
					return
				}
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

// GetNarration 根据章节ID获取章节解说（返回最新版本）
func (s *novelService) GetNarration(ctx context.Context, chapterID string) (*novel.Narration, error) {
	return s.narrationRepo.FindByChapterID(ctx, chapterID)
}

// GetNarrationByVersion 根据章节ID和版本号获取章节解说
func (s *novelService) GetNarrationByVersion(ctx context.Context, chapterID string, version int) (*novel.Narration, error) {
	return s.narrationRepo.FindByChapterIDAndVersion(ctx, chapterID, version)
}

// SetNarrationVersion 设置章节解说的版本号
func (s *novelService) SetNarrationVersion(ctx context.Context, narrationID string, version int) error {
	return s.narrationRepo.UpdateVersion(ctx, narrationID, version)
}

// GetNarrationVersions 获取章节的所有版本号
func (s *novelService) GetNarrationVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.narrationRepo.FindVersionsByChapterID(ctx, chapterID)
}

// GetAudioVersions 获取章节解说的所有音频版本号
func (s *novelService) GetAudioVersions(ctx context.Context, narrationID string) ([]int, error) {
	return s.audioRepo.FindVersionsByNarrationID(ctx, narrationID)
}

// GetSubtitleVersions 获取章节的所有字幕版本号
func (s *novelService) GetSubtitleVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.subtitleRepo.FindVersionsByChapterID(ctx, chapterID)
}

// GetImageVersions 获取章节的所有图片版本号
func (s *novelService) GetImageVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.imageRepo.FindVersionsByChapterID(ctx, chapterID)
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

// getNextNarrationVersion 获取章节的下一个版本号（自动递增）
// chapterID: 章节ID
// 例如：如果已有 1, 2，则返回 3
func (s *novelService) getNextNarrationVersion(ctx context.Context, chapterID string) (int, error) {
	versions, err := s.narrationRepo.FindVersionsByChapterID(ctx, chapterID)
	if err != nil {
		// 如果没有找到任何版本，返回 1
		return 1, nil
	}

	if len(versions) == 0 {
		return 1, nil
	}

	// 找到最大的版本号
	maxVersion := 0
	for _, v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}

	// 返回下一个版本号
	return maxVersion + 1, nil
}

// auditAndFilterNarration 对生成的章节解说内容进行审查和过滤（极度宽松模式）
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
