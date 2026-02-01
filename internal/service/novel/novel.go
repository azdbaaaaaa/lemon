package novel

import (
	"context"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/storage"
	novelrepo "lemon/internal/repository/novel"
	"lemon/internal/service"
)

// NovelService 小说服务接口
// 定义 novel 模块 service 层提供的能力
type NovelService interface {
	// CreateNovelFromResource 根据资源ID创建小说
	CreateNovelFromResource(ctx context.Context, resourceID, userID, workflowID string) (string, error)

	// SplitNovelIntoChapters 根据小说内容切分章节
	SplitNovelIntoChapters(ctx context.Context, novelID string, targetChapters int) error

	// GetNovel 获取小说信息
	GetNovel(ctx context.Context, novelID string) (*novel.Novel, error)

	// GetChapters 获取小说的所有章节
	GetChapters(ctx context.Context, novelID string) ([]*novel.Chapter, error)

	// GenerateNarrationForChapter 为单一章节生成解说文本
	GenerateNarrationForChapter(ctx context.Context, chapterID string) (string, error)

	// GenerateNarrationsForAllChapters 并发地为所有章节生成解说文本
	GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error

	// GetNarration 根据章节ID获取解说文案
	GetNarration(ctx context.Context, chapterID string) (*novel.Narration, error)

	// GenerateAudiosForNarration 为解说文案生成所有音频片段
	GenerateAudiosForNarration(ctx context.Context, narrationID string) ([]string, error)

	// GenerateSubtitlesForNarration 为解说文案生成字幕文件（ASS格式）
	// 需要先有音频记录（包含时间戳数据）
	GenerateSubtitlesForNarration(ctx context.Context, narrationID string) (string, error)
}

// novelService 小说服务实现
type novelService struct {
	resourceService service.ResourceService
	novelRepo       novelrepo.NovelRepository
	chapterRepo     novelrepo.ChapterRepository
	narrationRepo   novelrepo.NarrationRepository
	audioRepo       novelrepo.AudioRepository
	subtitleRepo    novelrepo.SubtitleRepository
	storage         storage.Storage
	llmProvider     noveltools.LLMProvider
	ttsProvider     noveltools.TTSProvider
}

// NewNovelService 创建小说服务
func NewNovelService(
	resourceService service.ResourceService,
	novelRepo novelrepo.NovelRepository,
	chapterRepo novelrepo.ChapterRepository,
	narrationRepo novelrepo.NarrationRepository,
	audioRepo novelrepo.AudioRepository,
	subtitleRepo novelrepo.SubtitleRepository,
	storage storage.Storage,
	llmProvider noveltools.LLMProvider,
	ttsProvider noveltools.TTSProvider,
) NovelService {
	return &novelService{
		resourceService: resourceService,
		novelRepo:       novelRepo,
		chapterRepo:     chapterRepo,
		narrationRepo:   narrationRepo,
		audioRepo:       audioRepo,
		subtitleRepo:    subtitleRepo,
		storage:         storage,
		llmProvider:     llmProvider,
		ttsProvider:     ttsProvider,
	}
}
