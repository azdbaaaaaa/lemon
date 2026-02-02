package novel

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/noveltools/providers"
	"lemon/internal/pkg/tts"
	novelrepo "lemon/internal/repository/novel"
	"lemon/internal/service"
)

// NovelService 小说服务接口
// 组合所有子模块的服务接口
type NovelService interface {
	ChapterService
	NarrationService
	AudioService
	SubtitleService
	ImageService
	CharacterService
	VideoService
}

// novelService 小说服务实现
type novelService struct {
	resourceService  service.ResourceService
	novelRepo        novelrepo.NovelRepository
	chapterRepo      novelrepo.ChapterRepository
	narrationRepo    novelrepo.ChapterNarrationRepository
	audioRepo        novelrepo.ChapterAudioRepository
	subtitleRepo     novelrepo.ChapterSubtitleRepository
	characterRepo    novelrepo.CharacterRepository
	chapterImageRepo novelrepo.ChapterImageRepository
	videoRepo        novelrepo.ChapterVideoRepository
	llmProvider      noveltools.LLMProvider
	ttsProvider      noveltools.TTSProvider
	imageProvider    noveltools.ImageProvider
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
	narrationRepo := novelrepo.NewChapterNarrationRepo(db)
	audioRepo := novelrepo.NewChapterAudioRepo(db)
	subtitleRepo := novelrepo.NewChapterSubtitleRepo(db)
	characterRepo := novelrepo.NewCharacterRepo(db)
	chapterImageRepo := novelrepo.NewChapterImageRepo(db)
	videoRepo := novelrepo.NewChapterVideoRepo(db)

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
		resourceService:  resourceService,
		novelRepo:        novelRepo,
		chapterRepo:      chapterRepo,
		narrationRepo:    narrationRepo,
		audioRepo:        audioRepo,
		subtitleRepo:     subtitleRepo,
		characterRepo:    characterRepo,
		chapterImageRepo: chapterImageRepo,
		videoRepo:        videoRepo,
		llmProvider:      llmProvider,
		ttsProvider:      ttsProvider,
		imageProvider:    imageProvider,
	}, nil
}
