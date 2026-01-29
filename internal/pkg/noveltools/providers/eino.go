package providers

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EinoProvider Eino 封装的 LLM 提供者（默认使用）
// 使用 ai/component 封装的 ChatModel（基于 eino-ext 的 ark 模块）
// 实现了 noveltools.LLMProvider 接口
type EinoProvider struct {
	chatModel model.ChatModel
}

// NewEinoProvider 创建基于 Eino 的 LLM 提供者（默认推荐使用）
//
// Args:
//   - chatModel: 通过 ai/component.NewChatModel 创建的 ChatModel 实例
//
// Returns:
//   - *EinoProvider: LLM 提供者实例
func NewEinoProvider(chatModel model.ChatModel) *EinoProvider {
	return &EinoProvider{
		chatModel: chatModel,
	}
}

// Generate 根据提示词生成文本（使用 eino ChatModel）
// 实现了 noveltools.LLMProvider 接口
func (p *EinoProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if p.chatModel == nil {
		return "", fmt.Errorf("chatModel is required")
	}

	// 构建消息
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	// 调用 ChatModel 的 Generate 方法
	response, err := p.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	// 提取内容
	content := response.Content
	if content == "" {
		return "", fmt.Errorf("empty response from chat model")
	}

	return content, nil
}

