package novel

import (
	"context"
	"fmt"
	"io"
	"strings"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/service"
)

// ChapterService 章节服务接口
// 定义小说和章节相关的能力
type ChapterService interface {
	// CreateNovelFromResource 根据资源ID创建小说
	CreateNovelFromResource(ctx context.Context, resourceID, userID string, narrationType novel.NarrationType, style novel.NovelStyle) (string, error)

	// SplitNovelIntoChapters 根据小说内容切分章节
	SplitNovelIntoChapters(ctx context.Context, novelID string, targetChapters int) error

	// GetNovel 获取小说信息
	GetNovel(ctx context.Context, novelID string) (*novel.Novel, error)

	// GetChapters 获取小说的所有章节
	GetChapters(ctx context.Context, novelID string) ([]*novel.Chapter, error)
}

// CreateNovelFromResource 第一步：根据资源ID获取小说内容，然后创建小说
// 返回创建的小说ID
func (s *novelService) CreateNovelFromResource(ctx context.Context, resourceID, userID string, narrationType novel.NarrationType, style novel.NovelStyle) (string, error) {
	// 使用 ResourceService 获取资源信息（系统内部请求，userID 为空）
	resResult, err := s.resourceService.GetResource(ctx, &service.GetResourceRequest{
		ResourceID: resourceID,
		UserID:     "", // 系统内部请求，可以访问所有资源
	})
	if err != nil {
		return "", fmt.Errorf("failed to find resource: %w", err)
	}
	res := resResult.Resource

	// 提取小说元数据（书名、作者、简介）
	title := res.Name // 默认使用文件名作为书名
	author := ""
	description := ""

	// 尝试从文件内容中提取元数据
	downloadReq := &service.DownloadFileRequest{
		UserID:     userID,
		ResourceID: res.ID,
	}
	downloadResult, err := s.resourceService.DownloadFile(ctx, downloadReq)
	if err == nil {
		defer downloadResult.Data.Close()
		// 读取前几行来提取元数据
		metadata := extractNovelMetadata(downloadResult.Data, res.Name)
		if metadata.Title != "" {
			title = metadata.Title
		}
		author = metadata.Author
		description = metadata.Description
	}

	novelID := id.New()
	novelEntity := &novel.Novel{
		ID:            novelID,
		ResourceID:    resourceID,
		UserID:        userID,
		Title:         title,
		Author:        author,
		Description:   description,
		NarrationType: narrationType,
		Style:         style,
	}

	if err := s.novelRepo.Create(ctx, novelEntity); err != nil {
		return "", fmt.Errorf("failed to create novel: %w", err)
	}

	return novelID, nil
}

// SplitNovelIntoChapters 第二步：根据小说内容切分章节，然后插入章节数据
// 需要先从资源中读取内容，然后切分并保存章节
func (s *novelService) SplitNovelIntoChapters(ctx context.Context, novelID string, targetChapters int) error {
	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("failed to find novel: %w", err)
	}

	// 使用 ResourceService 获取资源信息（系统内部请求，userID 为空）
	resResult, err := s.resourceService.GetResource(ctx, &service.GetResourceRequest{
		ResourceID: novelEntity.ResourceID,
		UserID:     "", // 系统内部请求，可以访问所有资源
	})
	if err != nil {
		return fmt.Errorf("failed to find resource: %w", err)
	}
	res := resResult.Resource

	// 通过 resource 模块下载文件
	downloadReq := &service.DownloadFileRequest{
		UserID:     novelEntity.UserID,
		ResourceID: res.ID,
	}
	downloadResult, err := s.resourceService.DownloadFile(ctx, downloadReq)
	if err != nil {
		return fmt.Errorf("failed to download resource: %w", err)
	}
	defer downloadResult.Data.Close()

	reader := downloadResult.Data

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

		// 计算章节统计信息
		totalChars := countChineseCharacters(seg.Text)
		wordCount := countChineseWords(seg.Text)
		lineCount := len(strings.Split(strings.TrimSpace(seg.Text), "\n"))

		chapterEntity := &novel.Chapter{
			ID:      chapterID,
			NovelID: novelID,
			UserID:  novelEntity.UserID,
			Sequence:    i + 1,
			Title:       seg.Title,
			ChapterText: seg.Text,
			TotalChars:  totalChars,
			WordCount:   wordCount,
			LineCount:   lineCount,
		}

		if err := s.chapterRepo.Create(ctx, chapterEntity); err != nil {
			return fmt.Errorf("failed to create chapter %d: %w", i+1, err)
		}
	}

	return nil
}

// countChineseCharacters 计算中文字符数量（包括中文标点）
func countChineseCharacters(text string) int {
	count := 0
	for _, r := range text {
		// 中文字符范围：\u4e00-\u9fff
		// 中文标点范围：\u3000-\u303f, \uff00-\uffef
		if (r >= 0x4e00 && r <= 0x9fff) || (r >= 0x3000 && r <= 0x303f) || (r >= 0xff00 && r <= 0xffef) {
			count++
		}
	}
	return count
}

// countChineseWords 计算中文字数（仅中文字符，不包括标点）
func countChineseWords(text string) int {
	count := 0
	for _, r := range text {
		// 仅计算中文字符，不包括标点
		if r >= 0x4e00 && r <= 0x9fff {
			count++
		}
	}
	return count
}

// GetNovel 获取小说信息
func (s *novelService) GetNovel(ctx context.Context, novelID string) (*novel.Novel, error) {
	return s.novelRepo.FindByID(ctx, novelID)
}

// GetChapters 获取小说的所有章节
func (s *novelService) GetChapters(ctx context.Context, novelID string) ([]*novel.Chapter, error) {
	return s.chapterRepo.FindByNovelID(ctx, novelID)
}

// NovelMetadata 小说元数据
type NovelMetadata struct {
	Title       string
	Author      string
	Description string
}

// extractNovelMetadata 从小说内容中提取元数据（书名、作者、简介）
// 读取文件的前几行，尝试解析格式如：
// 书名：xxx
// 作者：xxx
// 简介：xxx
func extractNovelMetadata(reader io.Reader, fileName string) NovelMetadata {
	metadata := NovelMetadata{
		Title: "", // 如果没有找到，使用文件名
	}

	// 读取前 10 行
	buf := make([]byte, 4096) // 读取前 4KB 内容
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return metadata
	}

	content := string(buf[:n])
	lines := strings.Split(content, "\n")

	// 解析前几行
	for i, line := range lines {
		if i >= 10 { // 只解析前 10 行
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析书名
		if strings.HasPrefix(line, "书名：") || strings.HasPrefix(line, "书名:") {
			metadata.Title = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "书名："), "书名:"))
			continue
		}

		// 解析作者
		if strings.HasPrefix(line, "作者：") || strings.HasPrefix(line, "作者:") {
			metadata.Author = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "作者："), "作者:"))
			continue
		}

		// 解析简介
		if strings.HasPrefix(line, "简介：") || strings.HasPrefix(line, "简介:") {
			// 简介可能跨多行，收集后续行直到遇到空行或下一个标签
			desc := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "简介："), "简介:"))
			// 继续读取后续行作为简介内容
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					break
				}
				// 如果下一行是新的标签，停止
				if strings.Contains(nextLine, "：") || strings.Contains(nextLine, ":") {
					break
				}
				if desc != "" {
					desc += " "
				}
				desc += nextLine
			}
			metadata.Description = desc
			break // 简介通常是最后一个，找到后可以停止
		}
	}

	// 如果没有找到书名，使用文件名（去掉扩展名）
	if metadata.Title == "" {
		// 去掉文件扩展名
		fileNameWithoutExt := fileName
		if idx := strings.LastIndex(fileName, "."); idx > 0 {
			fileNameWithoutExt = fileName[:idx]
		}
		metadata.Title = fileNameWithoutExt
	}

	return metadata
}
