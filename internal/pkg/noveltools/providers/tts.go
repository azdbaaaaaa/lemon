package providers

import (
	"context"

	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/tts"
)

// ByteDanceTTSProvider 字节跳动 TTS 提供者（使用 pkg/tts 的 Client）
// 实现了 noveltools.TTSProvider 接口
type ByteDanceTTSProvider struct {
	client *tts.Client
}

// NewByteDanceTTSProvider 创建基于 TTS 的提供者（使用 pkg/tts 的 Client）
//
// Args:
//   - client: TTS 客户端实例（通过 tts.NewClient 创建）
//
// Returns:
//   - *ByteDanceTTSProvider: TTS 提供者实例
func NewByteDanceTTSProvider(client *tts.Client) *ByteDanceTTSProvider {
	return &ByteDanceTTSProvider{
		client: client,
	}
}

// GenerateVoiceWithTimestamps 生成语音并获取时间戳（使用 TTS 客户端）
// 实现了 noveltools.TTSProvider 接口
func (p *ByteDanceTTSProvider) GenerateVoiceWithTimestamps(
	ctx context.Context,
	text string,
	audioPath string,
	speedRatio float64,
) (*noveltools.TTSResult, error) {
	if p.client == nil {
		return &noveltools.TTSResult{
			Success:      false,
			AudioPath:    audioPath,
			ErrorMessage: "TTS client is required",
		}, nil
	}

	// 调用 tts.Client，返回 tts.Result
	ttsResult, err := p.client.GenerateVoiceWithTimestamps(ctx, text, audioPath, speedRatio)
	if err != nil {
		return &noveltools.TTSResult{
			Success:      false,
			AudioPath:    audioPath,
			ErrorMessage: err.Error(),
		}, err
	}

	// 转换 tts.Result 到 noveltools.TTSResult
	result := &noveltools.TTSResult{
		Success:      ttsResult.Success,
		AudioPath:    ttsResult.AudioPath,
		ErrorMessage: ttsResult.ErrorMessage,
	}

	if ttsResult.TimestampData != nil {
		result.TimestampData = &noveltools.TimestampData{
			Text:                ttsResult.TimestampData.Text,
			AudioFile:           ttsResult.TimestampData.AudioFile,
			Duration:            ttsResult.TimestampData.Duration,
			CharacterTimestamps: convertCharTimestamps(ttsResult.TimestampData.CharacterTimestamps),
			GeneratedAt:         ttsResult.TimestampData.GeneratedAt,
		}
	}

	return result, nil
}

// convertCharTimestamps 转换字符时间戳
func convertCharTimestamps(ttsTimestamps []tts.CharTimestamp) []noveltools.CharTimestamp {
	result := make([]noveltools.CharTimestamp, len(ttsTimestamps))
	for i, ts := range ttsTimestamps {
		result[i] = noveltools.CharTimestamp{
			Character: ts.Character,
			StartTime: ts.StartTime,
			EndTime:   ts.EndTime,
		}
	}
	return result
}
