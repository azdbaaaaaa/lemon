package novel

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/noveltools/providers"
	"lemon/internal/pkg/tts"
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

	// GenerateImagesForNarration 为解说文案生成所有场景特写图片
	GenerateImagesForNarration(ctx context.Context, narrationID string) ([]string, error)

	// SyncCharactersFromNarration 从解说文案同步角色信息到小说级别
	SyncCharactersFromNarration(ctx context.Context, novelID, narrationID string) error

	// GetCharactersByNovelID 获取小说的所有角色
	GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error)

	// GetCharacterByName 根据名称获取角色
	GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error)
}

// novelService 小说服务实现
type novelService struct {
	resourceService    service.ResourceService
	novelRepo          novelrepo.NovelRepository
	chapterRepo        novelrepo.ChapterRepository
	narrationRepo      novelrepo.NarrationRepository
	audioRepo          novelrepo.AudioRepository
	subtitleRepo       novelrepo.SubtitleRepository
	characterRepo      novelrepo.CharacterRepository
	sceneShotImageRepo novelrepo.SceneShotImageRepository
	llmProvider        noveltools.LLMProvider
	ttsProvider        noveltools.TTSProvider
	imageProvider      noveltools.ImageProvider
}

// NewNovelService 创建小说服务
// 只需要传入必要的依赖，所有 repository 和 provider 在内部自动创建
func NewNovelService(
	db *mongo.Database,
	resourceService service.ResourceService,
) (NovelService, error) {
	// 初始化所有 repository
	novelRepo := novelrepo.NewNovelRepo(db)
	chapterRepo := novelrepo.NewChapterRepo(db)
	narrationRepo := novelrepo.NewNarrationRepo(db)
	audioRepo := novelrepo.NewAudioRepo(db)
	subtitleRepo := novelrepo.NewSubtitleRepo(db)
	characterRepo := novelrepo.NewCharacterRepo(db)
	sceneShotImageRepo := novelrepo.NewSceneShotImageRepo(db)

	// 初始化 LLM Provider（从环境变量读取配置）
	aiCfg := ark.ArkConfigFromEnv()
	arkClient, err := ark.NewLLMClient(aiCfg)
	if err != nil {
		return nil, fmt.Errorf("初始化 LLM Provider 失败: %w", err)
	}
	llmProvider := providers.NewArkProvider(arkClient)

	// 初始化 TTS Provider（从环境变量读取配置）
	ttsConfig := tts.ConfigFromEnv()
	ttsClient, err := tts.NewClient(ttsConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 TTS Provider 失败: %w", err)
	}
	ttsProvider := providers.NewByteDanceTTSProvider(ttsClient)

	// 初始化 Image Provider（从环境变量读取配置）
	// 使用 Ark 图片生成（使用官方 Go SDK）
	imageProvider, err := providers.NewArkImageProvider()
	if err != nil {
		return nil, fmt.Errorf("初始化 Image Provider 失败: %w", err)
	}

	return &novelService{
		resourceService:    resourceService,
		novelRepo:          novelRepo,
		chapterRepo:        chapterRepo,
		narrationRepo:      narrationRepo,
		audioRepo:          audioRepo,
		subtitleRepo:       subtitleRepo,
		characterRepo:      characterRepo,
		sceneShotImageRepo: sceneShotImageRepo,
		llmProvider:        llmProvider,
		ttsProvider:        ttsProvider,
		imageProvider:      imageProvider,
	}, nil
}
