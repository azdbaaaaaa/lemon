package novel

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

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

	// ListSubtitlesByNarration 获取解说的字幕列表（可指定版本；version<=0 则取最新版本）
	ListSubtitlesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Subtitle, int, error)

	// ListSubtitlesByShot 获取镜头的字幕列表
	ListSubtitlesByShot(ctx context.Context, shotID string) ([]*novel.Subtitle, error)

	// GenerateSubtitleForShot 为单个镜头生成字幕
	GenerateSubtitleForShot(ctx context.Context, shotID string) (string, error)
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
// GenerateSubtitlesForNarration 为章节解说生成所有字幕文件（ASS格式）
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GenerateSubtitlesForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
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

// GetSubtitleVersions 获取章节的所有字幕版本号
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GetSubtitleVersions(ctx context.Context, chapterID string) ([]int, error) {
	// TODO: 重构此方法，根据章节ID查询字幕版本号
	// 暂时返回空列表
	return []int{1}, nil
}

// getNextSubtitleVersion 获取章节的下一个字幕版本号（自动递增）
// chapterID: 章节ID
// baseVersion: 基础版本号（如 1），如果为0则自动生成下一个版本号
func (s *novelService) getNextSubtitleVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	// TODO: 重构此方法，根据 shot_id 或 chapter_id 查询字幕版本号
	// 暂时返回基础版本号或1
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

// ListSubtitlesByShot 获取镜头的字幕列表
func (s *novelService) ListSubtitlesByShot(ctx context.Context, shotID string) ([]*novel.Subtitle, error) {
	subtitles, err := s.subtitleRepo.FindByShotID(ctx, shotID)
	if err != nil {
		return nil, fmt.Errorf("find subtitles by shot_id: %w", err)
	}
	return subtitles, nil
}

// GenerateSubtitleForShot 为单个镜头生成字幕
func (s *novelService) GenerateSubtitleForShot(ctx context.Context, shotID string) (string, error) {
	// 1. 获取镜头信息
	shot, err := s.shotRepo.FindByID(ctx, shotID)
	if err != nil {
		return "", fmt.Errorf("find shot: %w", err)
	}

	// 2. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, shot.ChapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	// 3. 获取镜头的音频（需要包含时间戳）
	audios, err := s.audioRepo.FindByShotID(ctx, shotID)
	if err != nil {
		return "", fmt.Errorf("find audio: %w", err)
	}

	if len(audios) == 0 {
		return "", fmt.Errorf("shot %s has no audio", shotID)
	}

	// 使用最新的音频（按创建时间排序，取第一个）
	audio := audios[0]
	if len(audio.Timestamps) == 0 {
		return "", fmt.Errorf("audio %s has no timestamps", audio.ID)
	}

	// 4. 获取下一个字幕版本号
	nextVersion, err := s.getNextSubtitleVersion(ctx, chapter.ID, shot.Version)
	if err != nil {
		return "", fmt.Errorf("get next subtitle version: %w", err)
	}

	// 5. 将字符时间戳转换为段落时间戳
	segmentTimestamps := s.convertCharTimestampsToSegments(audio.Timestamps, audio.Text, audio.Duration)

	// 6. 生成ASS字幕内容
	assGenerator := noveltools.NewASSGenerator()
	assContent := assGenerator.GenerateASSContent(segmentTimestamps, fmt.Sprintf("Shot %d Subtitle", shot.Sequence))

	// 7. 上传字幕文件到资源服务
	uploadReq := &service.UploadFileRequest{
		UserID:      chapter.UserID,
		FileName:    fmt.Sprintf("subtitle_%s.ass", shotID),
		ContentType: "text/plain",
		Ext:         "ass",
		Data:        bytes.NewReader([]byte(assContent)),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload subtitle file: %w", err)
	}

	// 8. 创建字幕记录
	subtitleID := id.New()
	subtitle := &novel.Subtitle{
		ID:                 subtitleID,
		NovelID:            chapter.NovelID,
		UserID:             chapter.UserID,
		SubtitleType:       novel.SubtitleTypeShot,
		ShotID:             shot.ID,
		ChapterID:          chapter.ID,
		SubtitleResourceID: uploadResult.ResourceID,
		Format:             novel.SubtitleFormatASS,
		Prompt:             audio.Text,
		Version:            nextVersion,
		Status:             novel.TaskStatusCompleted,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.subtitleRepo.Create(ctx, subtitle); err != nil {
		return "", fmt.Errorf("create subtitle record: %w", err)
	}

	log.Info().
		Str("shot_id", shotID).
		Str("subtitle_id", subtitleID).
		Str("chapter_id", chapter.ID).
		Int("version", nextVersion).
		Msg("字幕生成成功")

	return subtitleID, nil
}

// convertCharTimestampsToSegments 将字符时间戳转换为段落时间戳
func (s *novelService) convertCharTimestampsToSegments(charTimestamps []novel.CharTime, text string, duration float64) []noveltools.SegmentTimestamp {
	if len(charTimestamps) == 0 {
		return []noveltools.SegmentTimestamp{}
	}

	// 将字符时间戳转换为 noveltools.CharTimestamp
	charTimestampsForCalc := make([]noveltools.CharTimestamp, 0, len(charTimestamps))
	for _, ct := range charTimestamps {
		charTimestampsForCalc = append(charTimestampsForCalc, noveltools.CharTimestamp{
			Character: ct.Character,
			StartTime: ct.StartTime,
			EndTime:   ct.EndTime,
		})
	}

	// 使用字幕时间戳计算器
	calculator := noveltools.NewSubtitleTimestampCalculator()
	
	// 将文本按标点符号分割成段落
	segments := s.splitTextIntoSegments(text)
	
	// 计算段落时间戳
	segmentTimestamps := calculator.CalculateSegmentTimestamps(segments, charTimestampsForCalc, text)

	// 根据音频时长调整时间戳
	adjustedTimestamps := adjustSubtitleTimestampsToAudioDuration(segmentTimestamps, duration)

	return adjustedTimestamps
}

// splitTextIntoSegments 将文本按标点符号分割成段落
func (s *novelService) splitTextIntoSegments(text string) []string {
	// 简单的分割：按句号、问号、感叹号分割
	segments := []string{}
	currentSegment := ""
	
	for _, char := range text {
		currentSegment += string(char)
		if char == '。' || char == '！' || char == '？' || char == '.' || char == '!' || char == '?' {
			if strings.TrimSpace(currentSegment) != "" {
				segments = append(segments, strings.TrimSpace(currentSegment))
			}
			currentSegment = ""
		}
	}
	
	// 添加最后一段（如果有）
	if strings.TrimSpace(currentSegment) != "" {
		segments = append(segments, strings.TrimSpace(currentSegment))
	}
	
	// 如果没有分割出段落，返回整个文本
	if len(segments) == 0 {
		segments = []string{text}
	}
	
	return segments
}
