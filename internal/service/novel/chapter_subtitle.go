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

// SubtitleService 章节字幕服务接口
// 定义章节字幕相关的能力
type SubtitleService interface {
	// GenerateSubtitlesForNarration 为章节解说生成所有字幕文件（ASS格式）
	// 为每个 narration shot 生成单独的字幕文件，与音频片段一一对应
	// 需要先有章节音频记录（包含时间戳数据）
	// 自动使用最新的版本号+1
	GenerateSubtitlesForNarration(ctx context.Context, narrationID string) ([]string, error)

	// GetSubtitleVersions 获取章节的所有字幕版本号
	GetSubtitleVersions(ctx context.Context, chapterID string) ([]int, error)
}

// GenerateSubtitlesForNarration 为章节解说生成所有字幕文件（ASS格式）
// 为每个 narration shot 生成单独的字幕文件，与音频片段一一对应
// 参考 Python 的 gen_ass.py 逻辑
//
// Args:
//   - ctx: 上下文
//   - narrationID: 章节解说ID
//
// Returns:
//   - []string: 生成的章节字幕ID列表
//   - error: 错误信息
func (s *novelService) GenerateSubtitlesForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// 1. 从数据库获取章节解说
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find narration: %w", err)
	}

	if narration.Content == nil {
		return nil, fmt.Errorf("narration content is nil")
	}

	// 2. 自动生成下一个版本号（基于章节ID，独立递增）
	subtitleVersion, err := s.getNextSubtitleVersion(ctx, narration.ChapterID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get next subtitle version: %w", err)
	}

	// 3. 获取该章节解说的最新版本的音频记录（需要时间戳数据）
	// 先获取所有版本号，找到最新版本
	audioVersions, err := s.audioRepo.FindVersionsByNarrationID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find audio versions: %w", err)
	}

	if len(audioVersions) == 0 {
		return nil, fmt.Errorf("no audio records found for narration %s, please generate audio first", narrationID)
	}

	// 找到最新版本号
	maxAudioVersion := 0
	for _, v := range audioVersions {
		if v > maxAudioVersion {
			maxAudioVersion = v
		}
	}

	// 只获取最新版本的音频
	audios, err := s.audioRepo.FindByNarrationIDAndVersion(ctx, narrationID, maxAudioVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to find audios: %w", err)
	}

	if len(audios) == 0 {
		return nil, fmt.Errorf("no audio records found for narration %s version %d, please generate audio first", narrationID, maxAudioVersion)
	}

	// 4. 从章节解说中提取所有文本
	extractor := noveltools.NewNarrationExtractor()
	narrationTexts, err := extractor.ExtractNarrationTexts(narration.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract narration texts: %w", err)
	}

	if len(narrationTexts) == 0 {
		return nil, fmt.Errorf("no narration texts found")
	}

	// 5. 为每个音频片段生成对应的字幕文件
	var subtitleIDs []string
	for i, audio := range audios {
		sequence := audio.Sequence
		if sequence == 0 {
			sequence = i + 1 // 如果没有 sequence，使用索引+1
		}

		// 获取对应的文本
		var narrationText string
		if i < len(narrationTexts) {
			narrationText = narrationTexts[i]
		} else {
			narrationText = audio.Text // 如果没有对应的文本，使用音频记录的文本
		}

		if narrationText == "" {
			log.Warn().Int("sequence", sequence).Msg("解说文本为空，跳过字幕生成")
			continue
		}

		// 生成单个字幕文件
		subtitleID, err := s.generateSingleSubtitle(ctx, narration, audio, sequence, narrationText, subtitleVersion)
		if err != nil {
			log.Error().Err(err).Int("sequence", sequence).Msg("生成字幕失败")
			return nil, fmt.Errorf("failed to generate subtitle for sequence %d: %w", sequence, err)
		}

		subtitleIDs = append(subtitleIDs, subtitleID)
	}

	return subtitleIDs, nil
}

// generateSingleSubtitle 为单个音频片段生成字幕文件
func (s *novelService) generateSingleSubtitle(
	ctx context.Context,
	narration *novel.ChapterNarration,
	audio *novel.ChapterAudio,
	sequence int,
	narrationText string,
	version int,
) (string, error) {
	// 1. 检查音频是否有时间戳数据
	if len(audio.Timestamps) == 0 {
		return "", fmt.Errorf("audio record has no timestamps, sequence=%d", sequence)
	}

	// 2. 转换字符时间戳（不需要时间偏移，因为每个字幕从0开始）
	characterTimestamps := make([]noveltools.CharTimestamp, 0, len(audio.Timestamps))
	for _, charTime := range audio.Timestamps {
		characterTimestamps = append(characterTimestamps, noveltools.CharTimestamp{
			Character: charTime.Character,
			StartTime: charTime.StartTime, // 从0开始，不需要偏移
			EndTime:   charTime.EndTime,
		})
	}

	// 3. 使用 SubtitleSplitter 分割文本（每段最大20字符，避免字幕片段过短）
	maxLength := 20
	splitter := noveltools.NewSubtitleSplitter(maxLength)
	segments := splitter.SplitTextNaturally(narrationText)

	if len(segments) == 0 {
		return "", fmt.Errorf("no segments found after splitting text, sequence=%d", sequence)
	}

	// 4. 使用 SubtitleTimestampCalculator 计算时间戳
	calculator := noveltools.NewSubtitleTimestampCalculator()
	segmentTimestamps := calculator.CalculateSegmentTimestamps(
		segments,
		characterTimestamps,
		narrationText,
	)

	if len(segmentTimestamps) == 0 {
		return "", fmt.Errorf("failed to calculate segment timestamps, sequence=%d", sequence)
	}

	// 4.5. 根据音频时长调整字幕时间戳（确保字幕时长不超过音频时长）
	// 参考 Python 版本：字幕时间戳应该基于音频的实际时长
	audioDuration := audio.Duration
	if audioDuration <= 0 {
		// 如果音频时长为 0，尝试从时间戳数据中获取
		if len(characterTimestamps) > 0 {
			lastCharTime := characterTimestamps[len(characterTimestamps)-1]
			audioDuration = lastCharTime.EndTime
		}
		if audioDuration <= 0 {
			audioDuration = 10.0 // 默认值
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", sequence).
				Msg("音频 duration 为 0，使用默认值 10 秒")
		}
	}

	// 调整字幕时间戳，确保不超过音频时长
	segmentTimestamps = adjustSubtitleTimestampsToAudioDuration(segmentTimestamps, audioDuration)

	// 5. 使用 ASSGenerator 生成 ASS 内容
	assGenerator := noveltools.NewASSGenerator()
	title := fmt.Sprintf("Narration Subtitle %d", sequence)
	assContent := assGenerator.GenerateASSContent(segmentTimestamps, title)

	// 6. 直接在内存中创建 ASS 文件的 io.Reader
	assContentBytes := []byte(assContent)
	assReader := bytes.NewReader(assContentBytes)

	// 7. 通过 resource 模块上传 ASS 文件
	userID := narration.UserID
	fileName := fmt.Sprintf("%s_subtitle_%02d.ass", narration.ID, sequence)
	contentType := "text/x-ass"
	ext := "ass"

	uploadReq := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    fileName,
		ContentType: contentType,
		Ext:         ext,
		Data:        assReader,
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload ASS file via resource service: %w", err)
	}

	resourceID := uploadResult.ResourceID

	// 8. 构建章节字幕生成参数提示词
	subtitlePrompt := fmt.Sprintf("字幕生成参数: maxLength=%d, format=ass, segmentCount=%d", maxLength, len(segmentTimestamps))

	// 9. 创建 chapter_subtitle 记录
	subtitleID := id.New()
	subtitleEntity := &novel.ChapterSubtitle{
		ID:                 subtitleID,
		ChapterID:          narration.ChapterID,
		NarrationID:        narration.ID,
		UserID:             narration.UserID,
		Sequence:           sequence,
		SubtitleResourceID: resourceID,
		Format:             "ass",
		Prompt:             subtitlePrompt,
		Version:            version,
		Status:             "completed",
	}

	if err := s.subtitleRepo.Create(ctx, subtitleEntity); err != nil {
		return "", fmt.Errorf("failed to create subtitle record: %w", err)
	}

	return subtitleID, nil
}

// adjustSubtitleTimestampsToAudioDuration 根据音频时长调整字幕时间戳
// 确保字幕的最后一个时间戳不超过音频时长
// 参考 Python 版本：字幕时间戳应该严格基于音频的实际时长
func adjustSubtitleTimestampsToAudioDuration(segmentTimestamps []noveltools.SegmentTimestamp, audioDuration float64) []noveltools.SegmentTimestamp {
	if len(segmentTimestamps) == 0 {
		return segmentTimestamps
	}

	// 如果最后一个字幕的结束时间已经小于等于音频时长，不需要调整
	lastEndTime := segmentTimestamps[len(segmentTimestamps)-1].EndTime
	if lastEndTime <= audioDuration {
		return segmentTimestamps
	}

	// 如果字幕总时长超过音频时长，需要按比例压缩
	// 计算压缩比例
	scaleFactor := audioDuration / lastEndTime

	// 按比例压缩所有时间戳
	adjusted := make([]noveltools.SegmentTimestamp, len(segmentTimestamps))
	for i, seg := range segmentTimestamps {
		adjusted[i] = noveltools.SegmentTimestamp{
			Text:      seg.Text,
			StartTime: seg.StartTime * scaleFactor,
			EndTime:   seg.EndTime * scaleFactor,
		}
	}

	// 确保最后一个字幕的结束时间正好等于音频时长（避免浮点数误差）
	if len(adjusted) > 0 {
		adjusted[len(adjusted)-1].EndTime = audioDuration
		// 确保最后一个字幕的开始时间不超过结束时间
		if adjusted[len(adjusted)-1].StartTime >= adjusted[len(adjusted)-1].EndTime {
			adjusted[len(adjusted)-1].StartTime = adjusted[len(adjusted)-1].EndTime - 0.5
			if adjusted[len(adjusted)-1].StartTime < 0 {
				adjusted[len(adjusted)-1].StartTime = 0
			}
		}
	}

	log.Info().
		Float64("original_last_time", lastEndTime).
		Float64("audio_duration", audioDuration).
		Float64("scale_factor", scaleFactor).
		Float64("adjusted_last_time", adjusted[len(adjusted)-1].EndTime).
		Msg("字幕时间戳已根据音频时长调整")

	return adjusted
}

// getNextSubtitleVersion 获取章节的下一个字幕版本号（自动递增）
// chapterID: 章节ID
// baseVersion: 基础版本号（如 1），如果为0则自动生成下一个版本号
func (s *novelService) getNextSubtitleVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	versions, err := s.subtitleRepo.FindVersionsByChapterID(ctx, chapterID)
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
