package novel

import (
	"context"
	"fmt"

	"lemon/internal/model/novel"
)

// ListAudiosByNarration 获取解说的音频列表（可指定版本；version<=0 则取最新版本）
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) ListAudiosByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Audio, int, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, 0, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// ListSubtitlesByNarration 获取解说的字幕列表（可指定版本；version<=0 则取最新版本）
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) ListSubtitlesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Subtitle, int, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, 0, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// ListImagesByNarration 获取解说的图片列表（可指定版本；version<=0 则取最新版本）
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) ListImagesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Image, int, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, 0, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// ListVideosByChapter 获取章节视频列表（可指定版本；version<=0 则取最新版本）
func (s *novelService) ListVideosByChapter(ctx context.Context, chapterID string, version int) ([]*novel.Video, int, error) {
	// 直接查询章节的所有视频
	allVideos, err := s.videoRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		return nil, 0, fmt.Errorf("find videos: %w", err)
	}

	// 如果指定了版本，过滤版本
	if version > 0 {
		filtered := make([]*novel.Video, 0)
		for _, video := range allVideos {
			if video.Version == version {
				filtered = append(filtered, video)
			}
		}
		allVideos = filtered
	} else {
		// 否则取最新版本
		// 找到最大版本号
		maxVersion := 0
		for _, video := range allVideos {
			if video.Version > maxVersion {
				maxVersion = video.Version
			}
		}
		// 只保留最新版本的视频
		filtered := make([]*novel.Video, 0)
		for _, video := range allVideos {
			if video.Version == maxVersion {
				filtered = append(filtered, video)
			}
		}
		allVideos = filtered
	}

	// 确定返回的版本号
	returnVersion := 0
	if len(allVideos) > 0 {
		returnVersion = allVideos[0].Version
	}

	return allVideos, returnVersion, nil
}
