package novel

import (
	"bytes"
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/service"
)

// AudioService 章节音频服务接口
// 定义章节音频相关的能力
type AudioService interface {
	// GenerateAudiosForNarration 为章节解说生成所有章节音频片段
	// 自动使用最新的版本号+1
	GenerateAudiosForNarration(ctx context.Context, narrationID string) ([]string, error)

	// GetAudioVersions 获取章节解说的所有音频版本号
	GetAudioVersions(ctx context.Context, narrationID string) ([]int, error)
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
func (s *novelService) GenerateAudiosForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// 1. 从数据库获取章节解说
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find narration: %w", err)
	}

	// 2. 从独立的表中查询所有镜头（按 index 排序）
	shots, err := s.shotRepo.FindByNarrationID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find shots: %w", err)
	}

	if len(shots) == 0 {
		return nil, fmt.Errorf("no shots found for narration")
	}

	// 3. 自动生成下一个版本号（基于章节ID，独立递增）
	audioVersion, err := s.getNextAudioVersion(ctx, narration.ChapterID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get next audio version: %w", err)
	}

	// 4. 从 Shot 表中提取所有解说文本（按 index 排序）
	var narrationTexts []string
	for _, shot := range shots {
		if shot.Narration != "" {
			narrationTexts = append(narrationTexts, shot.Narration)
		}
	}

	if len(narrationTexts) == 0 {
		return nil, fmt.Errorf("no narration texts found")
	}

	// 3. 为每段解说文本生成章节音频
	textCleaner := noveltools.NewTextCleaner()
	var audioIDs []string
	for i, narrationText := range narrationTexts {
		sequence := i + 1

		// 清理文本用于TTS
		cleanText := textCleaner.CleanTextForTTS(narrationText)
		if cleanText == "" {
			log.Warn().Int("sequence", sequence).Msg("清理后的文本为空，跳过")
			continue
		}

		// 生成章节音频
		audioID, err := s.generateSingleAudio(ctx, narration, sequence, cleanText, audioVersion)
		if err != nil {
			log.Error().Err(err).Int("sequence", sequence).Msg("生成章节音频失败")
			return nil, fmt.Errorf("failed to generate audio for sequence %d: %w", sequence, err)
		}

		audioIDs = append(audioIDs, audioID)
	}

	return audioIDs, nil
}

// generateSingleAudio 生成单个章节音频片段
func (s *novelService) generateSingleAudio(
	ctx context.Context,
	narration *novel.Narration,
	sequence int,
	text string,
	version int,
) (string, error) {
	// 1. 调用 TTS Provider 生成音频（1.2倍速，参考 Python 脚本）
	speedRatio := 1.2
	ttsResult, err := s.ttsProvider.GenerateVoiceWithTimestamps(ctx, text, speedRatio)
	if err != nil {
		return "", fmt.Errorf("TTS generation failed: %w", err)
	}

	if !ttsResult.Success {
		return "", fmt.Errorf("TTS generation failed: %s", ttsResult.ErrorMessage)
	}

	// 构建 TTS 参数提示词（记录生成参数）
	ttsPrompt := fmt.Sprintf("TTS参数: speedRatio=%.2f, textLength=%d", speedRatio, len(text))

	// 2. 通过 resource 模块上传音频文件（直接使用返回的音频数据）
	userID := narration.UserID
	fileName := fmt.Sprintf("%s_audio_%02d.mp3", narration.ID, sequence)
	contentType := "audio/mpeg"
	ext := "mp3"

	uploadReq := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    fileName,
		ContentType: contentType,
		Ext:         ext,
		Data:        bytes.NewReader(ttsResult.AudioData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload audio file via resource service: %w", err)
	}

	resourceID := uploadResult.ResourceID

	// 3. 转换时间戳数据
	charTimes := make([]novel.CharTime, 0, len(ttsResult.TimestampData.CharacterTimestamps))
	for _, ts := range ttsResult.TimestampData.CharacterTimestamps {
		charTimes = append(charTimes, novel.CharTime{
			Character: ts.Character,
			StartTime: ts.StartTime,
			EndTime:   ts.EndTime,
		})
	}

	// 4. 获取音频时长（使用 TTS API 返回的真实时长）
	audioDuration := ttsResult.Duration
	if audioDuration <= 0 {
		// 如果 Duration 为 0，尝试从 TimestampData 获取
		if ttsResult.TimestampData != nil && ttsResult.TimestampData.Duration > 0 {
			audioDuration = ttsResult.TimestampData.Duration
		} else {
			// 降级方案：如果都获取不到，使用默认值 10 秒
			audioDuration = 10.0
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", sequence).
				Msg("TTS API 返回的 duration 为 0，使用默认值 10 秒")
		}
	}

	// 8. 创建 chapter_audio 记录
	audioID := id.New()
	audioEntity := &novel.Audio{
		ID:              audioID,
		NarrationID:     narration.ID,
		ChapterID:       narration.ChapterID,
		UserID:          narration.UserID,
		Sequence:        sequence,
		AudioResourceID: resourceID,
		Duration:        audioDuration,
		Text:            text,
		Timestamps:      charTimes,
		Prompt:          ttsPrompt,
		Version:         version, // 使用指定的版本号
		Status:          "completed",
	}

	if err := s.audioRepo.Create(ctx, audioEntity); err != nil {
		return "", fmt.Errorf("failed to create audio record: %w", err)
	}

	return audioID, nil
}

// getNextAudioVersion 获取章节的下一个音频版本号（自动递增）
// chapterID: 章节ID
// baseVersion: 基础版本号（如 1），如果为0则自动生成下一个版本号
func (s *novelService) getNextAudioVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	versions, err := s.audioRepo.FindVersionsByChapterID(ctx, chapterID)
	if err != nil {
		// 如果没有找到任何版本，返回 1 或基础版本号
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	if len(versions) == 0 {
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	// 如果指定了基础版本号，检查该版本是否已存在
	if baseVersion > 0 {
		for _, v := range versions {
			if v == baseVersion {
				// 该版本已存在，返回下一个版本号
				maxVersion := 0
				for _, v := range versions {
					if v > maxVersion {
						maxVersion = v
					}
				}
				return maxVersion + 1, nil
			}
		}
		// 该版本不存在，直接返回
		return baseVersion, nil
	}

	// 如果没有指定基础版本号，查找所有版本号中的最大值
	maxVersion := 0
	for _, v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}

	return maxVersion + 1, nil
}
