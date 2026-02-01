package providers

import (
	"context"

	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/noveltools"
)

// ByteDanceTTSProvider 字节跳动 TTS 提供者（使用 pkg/ark 的 TTSClient）
// 实现了 noveltools.TTSProvider 接口
type ByteDanceTTSProvider struct {
	client *ark.TTSClient
}

// NewByteDanceTTSProvider 创建基于 Ark TTS 的提供者（使用 pkg/ark 的 TTSClient）
//
// Args:
//   - client: TTS 客户端实例（通过 ark.NewTTSClient 创建）
//
// Returns:
//   - *ByteDanceTTSProvider: TTS 提供者实例
func NewByteDanceTTSProvider(client *ark.TTSClient) *ByteDanceTTSProvider {
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
	return p.client.GenerateVoiceWithTimestamps(ctx, text, audioPath, speedRatio)
}
