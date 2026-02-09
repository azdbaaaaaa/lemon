package novel

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/service"

	"github.com/rs/zerolog/log"
)

// AudioService 章节音频服务接口
// 定义章节音频相关的能力
type AudioService interface {
	// GenerateAudiosForNarration 为章节解说生成所有章节音频片段
	// 自动使用最新的版本号+1
	GenerateAudiosForNarration(ctx context.Context, narrationID string) ([]string, error)

	// GenerateAudiosForChapter 为章节的所有 shot 生成音频
	// 基于章节的 active_scene_version 获取所有 shot，为每个 shot 的 narration 文本生成音频
	GenerateAudiosForChapter(ctx context.Context, chapterID string) ([]string, error)

	// GetAudioVersions 获取章节解说的所有音频版本号
	GetAudioVersions(ctx context.Context, narrationID string) ([]int, error)

	// ListAudiosByNarration 获取解说的音频列表（可指定版本；version<=0 则取最新版本）
	ListAudiosByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Audio, int, error)

	// ListAudiosByChapter 获取章节的音频列表（可指定版本；version<=0 则取最新版本）
	ListAudiosByChapter(ctx context.Context, chapterID string, version int) ([]*novel.Audio, int, error)
}

// GenerateAudiosForNarration 为章节解说生成所有章节音频片段
// 参考 Python 的 gen_audio.py 逻辑
//
// Args:
//   - ctx: 上下文
//   - narrationID: 章节解说ID
//   - version: 音频版本号，如果为空则使用章节解说的版本号，如果指定则自动生成下一个版本号
//
// Returns:
//   - []string: 生成的章节音频ID列表
//   - error: 错误信息
//
// GenerateAudiosForNarration 为章节解说生成所有章节音频片段
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GenerateAudiosForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// GetAudioVersions 获取章节解说的所有音频版本号
// 注意：narration 模块已删除，此方法暂时返回错误
func (s *novelService) GetAudioVersions(ctx context.Context, narrationID string) ([]int, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// GenerateAudiosForChapter 为章节的所有 shot 生成音频
// 基于章节的 active_scene_version 获取所有 shot，为每个 shot 的 narration 文本生成音频
func (s *novelService) GenerateAudiosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	if chapter.ActiveSceneVersion == 0 {
		return nil, fmt.Errorf("chapter has no active scene version")
	}

	// 2. 获取该版本的所有 shot
	shots, err := s.shotRepo.FindByChapterIDAndVersion(ctx, chapterID, chapter.ActiveSceneVersion)
	if err != nil {
		return nil, fmt.Errorf("find shots: %w", err)
	}

	if len(shots) == 0 {
		return nil, fmt.Errorf("no shots found for chapter %s, version %d", chapterID, chapter.ActiveSceneVersion)
	}

	// 3. 获取下一个音频版本号
	nextVersion, err := s.getNextAudioVersion(ctx, chapterID, chapter.ActiveSceneVersion)
	if err != nil {
		return nil, fmt.Errorf("get next audio version: %w", err)
	}

	// 4. 异步生成音频（类似图片生成）
	go func() {
		ctx := context.Background()
		for _, shot := range shots {
			_, err := s.generateAudioForShot(ctx, shot, chapter, nextVersion)
			if err != nil {
				log.Error().
					Err(err).
					Str("shot_id", shot.ID).
					Str("chapter_id", chapterID).
					Msg("生成音频失败")
				continue
			}
		}
	}()

	// 5. 立即返回（异步执行）
	return []string{}, nil
}

// generateAudioForShot 为单个 shot 生成音频
func (s *novelService) generateAudioForShot(ctx context.Context, shot *novel.Shot, chapter *novel.Chapter, version int) (string, error) {
	// 检查 shot 是否有 narration 文本
	if shot.Narration == "" {
		return "", fmt.Errorf("shot %s has no narration text", shot.ID)
	}

	// 1. 使用 TTS 生成音频
	ttsResult, err := s.ttsProvider.GenerateVoiceWithTimestamps(ctx, shot.Narration, 1.0)
	if err != nil {
		return "", fmt.Errorf("generate voice: %w", err)
	}

	if !ttsResult.Success {
		return "", fmt.Errorf("TTS generation failed: %s", ttsResult.ErrorMessage)
	}

	// 2. 上传音频文件到资源服务
	uploadReq := service.UploadFileRequest{
		FileName:    fmt.Sprintf("audio_%s.mp3", id.New()),
		ContentType: "audio/mpeg",
		Ext:         "mp3",
		Data:        bytes.NewReader(ttsResult.AudioData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, &uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload audio: %w", err)
	}

	// 3. 转换时间戳
	charTimestamps := make([]novel.CharTime, 0, len(ttsResult.TimestampData.CharacterTimestamps))
	if ttsResult.TimestampData != nil {
		for _, ts := range ttsResult.TimestampData.CharacterTimestamps {
			charTimestamps = append(charTimestamps, novel.CharTime{
				Character: ts.Character,
				StartTime: ts.StartTime,
				EndTime:   ts.EndTime,
			})
		}
	}

	// 4. 创建音频记录
	audioID := id.New()
	audio := &novel.Audio{
		ID:              audioID,
		NovelID:         chapter.NovelID,
		UserID:          chapter.UserID,
		AudioType:       novel.AudioTypeShot,
		ShotID:          shot.ID,
		ChapterID:       chapter.ID,
		AudioResourceID: uploadResult.ResourceID,
		Duration:        ttsResult.Duration,
		Text:            shot.Narration,
		Timestamps:      charTimestamps,
		Version:         version,
		Status:          novel.TaskStatusCompleted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.audioRepo.Create(ctx, audio); err != nil {
		return "", fmt.Errorf("create audio record: %w", err)
	}

	log.Info().
		Str("shot_id", shot.ID).
		Str("audio_id", audioID).
		Str("chapter_id", chapter.ID).
		Float64("duration", ttsResult.Duration).
		Msg("音频生成成功")

	return audioID, nil
}

// getNextAudioVersion 获取章节的下一个音频版本号（自动递增）
// chapterID: 章节ID
// baseVersion: 基础版本号（如 1），如果为0则自动生成下一个版本号
func (s *novelService) getNextAudioVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	// 查询该章节已有的音频版本号
	audios, err := s.audioRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		// 如果查询失败，返回基础版本号
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	// 找到该章节的最大版本号
	maxVersion := 0
	for _, audio := range audios {
		if audio.Version > maxVersion {
			maxVersion = audio.Version
		}
	}

	// 如果指定了基础版本号，使用它；否则使用最大版本号+1
	if baseVersion > 0 {
		if baseVersion > maxVersion {
			return baseVersion, nil
		}
		return maxVersion + 1, nil
	}

	if maxVersion == 0 {
		return 1, nil
	}
	return maxVersion + 1, nil
}

// ListAudiosByChapter 获取章节的音频列表（可指定版本；version<=0 则取最新版本）
func (s *novelService) ListAudiosByChapter(ctx context.Context, chapterID string, version int) ([]*novel.Audio, int, error) {
	// 获取该章节的所有 shot
	shots, err := s.shotRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		return nil, 0, fmt.Errorf("find shots: %w", err)
	}

	if len(shots) == 0 {
		return []*novel.Audio{}, 0, nil
	}

	// 获取所有 shot 的音频
	allAudios := make([]*novel.Audio, 0)
	for _, shot := range shots {
		audios, err := s.audioRepo.FindByShotID(ctx, shot.ID)
		if err != nil {
			continue
		}
		allAudios = append(allAudios, audios...)
	}

	// 如果指定了版本，过滤版本
	if version > 0 {
		filtered := make([]*novel.Audio, 0)
		for _, audio := range allAudios {
			if audio.Version == version {
				filtered = append(filtered, audio)
			}
		}
		allAudios = filtered
	} else {
		// 否则取最新版本
		// 找到最大版本号
		maxVersion := 0
		for _, audio := range allAudios {
			if audio.Version > maxVersion {
				maxVersion = audio.Version
			}
		}
		// 只保留最新版本的音频
		filtered := make([]*novel.Audio, 0)
		for _, audio := range allAudios {
			if audio.Version == maxVersion {
				filtered = append(filtered, audio)
			}
		}
		allAudios = filtered
	}

	// 确定返回的版本号
	returnVersion := 0
	if len(allAudios) > 0 {
		returnVersion = allAudios[0].Version
	}

	return allAudios, returnVersion, nil
}
