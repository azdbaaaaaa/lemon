package novel

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/noveltools"
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
