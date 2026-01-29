package providers

import (
	"context"
	"fmt"

	"lemon/internal/pkg/ark"
)

// ArkProvider Ark 实现的 LLM 提供者（使用 pkg/ark 的 Client）
// 实现了 noveltools.LLMProvider 接口
// 注意：推荐使用 EinoProvider（基于 eino-ext），此实现保留用于向后兼容
type ArkProvider struct {
	client *ark.Client
}

// NewArkProvider 创建基于 Ark 的 LLM 提供者（使用 pkg/ark 的 Client）
//
// Args:
//   - client: Ark 客户端实例（通过 ark.NewClient 创建）
//
// Returns:
//   - *ArkProvider: LLM 提供者实例
func NewArkProvider(client *ark.Client) *ArkProvider {
	return &ArkProvider{
		client: client,
	}
}

// Generate 根据提示词生成文本（使用 Ark 客户端）
// 实现了 noveltools.LLMProvider 接口
func (p *ArkProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("ark client is required")
	}
	return p.client.CreateChatCompletionSimple(ctx, prompt)
}
