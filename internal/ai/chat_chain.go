package ai

import (
	"context"

	"lemon/internal/config"
	"lemon/internal/model"
)

// ChatChain 对话链 - 封装 Eino Chain
// 职责: LLM 对话能力，消息格式转换
type ChatChain struct {
	cfg *config.AIConfig
	// chain compose.Runnable[string, *schema.Message] // Eino Chain (后续实现)
}

// NewChatChain 创建对话链
func NewChatChain(cfg *config.AIConfig) (*ChatChain, error) {
	// TODO: 初始化 Eino Chain
	// 1. 创建 ChatModel (OpenAI/Azure/Claude)
	// 2. 创建 PromptTemplate
	// 3. 组装 Chain

	return &ChatChain{
		cfg: cfg,
	}, nil
}

// Run 同步执行对话
func (c *ChatChain) Run(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// TODO: 集成 Eino 实现真正的 AI 对话
	// 当前返回模拟响应

	// 模拟响应
	content := "Hello! This is a mock response. "
	if len(req.History) > 0 {
		content += "I can see you have conversation history. "
	}
	content += "Eino integration coming soon."

	return &ChatResponse{
		Content: content,
		Usage: &model.TokenUsage{
			PromptTokens:     countTokens(req.Message),
			CompletionTokens: countTokens(content),
			TotalTokens:      countTokens(req.Message) + countTokens(content),
		},
	}, nil
}

// Stream 流式执行对话
func (c *ChatChain) Stream(ctx context.Context, req *ChatRequest) (<-chan *model.ChatChunk, error) {
	// TODO: 集成 Eino 实现流式响应

	ch := make(chan *model.ChatChunk, 10)

	go func() {
		defer close(ch)

		// 模拟流式响应
		chunks := []string{"Hello! ", "This is ", "a streaming ", "response. ", "Eino coming soon."}
		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			case ch <- &model.ChatChunk{Content: chunk}:
			}
		}

		// 发送完成信号
		ch <- &model.ChatChunk{
			Done: true,
			Usage: &model.TokenUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
	}()

	return ch, nil
}

// countTokens 简单的 token 计数 (实际应使用 tiktoken)
func countTokens(text string) int {
	// 简单估算: 4个字符 ≈ 1个token
	return len(text) / 4
}
