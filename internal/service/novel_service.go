package service

import (
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/storage"
	novelrepo "lemon/internal/repository/novel"
)

// NovelService 小说服务
// 用途：对单一章节生成解说，并落库到 narrations 表；根据解说文案生成音频和字幕
type NovelService struct {
	resourceService ResourceService
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
	resourceService ResourceService,
	novelRepo novelrepo.NovelRepository,
	chapterRepo novelrepo.ChapterRepository,
	narrationRepo novelrepo.NarrationRepository,
	audioRepo novelrepo.AudioRepository,
	subtitleRepo novelrepo.SubtitleRepository,
	storage storage.Storage,
	llmProvider noveltools.LLMProvider,
	ttsProvider noveltools.TTSProvider,
) *NovelService {
	return &NovelService{
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
