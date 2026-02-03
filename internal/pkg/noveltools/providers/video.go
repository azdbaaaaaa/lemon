package providers

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/noveltools"
)

// ArkVideoProvider Ark 视频生成提供者
// 适配层，调用 ark.ArkVideoClient
type ArkVideoProvider struct {
	client *ark.ArkVideoClient
}

// NewArkVideoProvider 创建 Ark 视频生成提供者
// 从环境变量读取配置，创建 ark.ArkVideoClient
func NewArkVideoProvider() (noveltools.VideoProvider, error) {
	config := ark.ArkVideoConfigFromEnv()
	client, err := ark.NewArkVideoClient(config)
	if err != nil {
		return nil, fmt.Errorf("create Ark Video client: %w", err)
	}

	return &ArkVideoProvider{
		client: client,
	}, nil
}

// GenerateVideoFromImage 从图片生成视频
// 调用 ark.ArkVideoClient.GenerateVideoFromImage
func (p *ArkVideoProvider) GenerateVideoFromImage(ctx context.Context, imageDataURL string, duration int, prompt string) ([]byte, error) {
	videoData, err := p.client.GenerateVideoFromImage(ctx, imageDataURL, duration, prompt)
	if err != nil {
		return nil, fmt.Errorf("Ark generate video: %w", err)
	}

	log.Info().
		Int("duration", duration).
		Int("size", len(videoData)).
		Str("prompt", prompt).
		Msg("Ark 视频生成成功")

	return videoData, nil
}
