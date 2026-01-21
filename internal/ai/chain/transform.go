package chain

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"lemon/internal/ai/component"
	"lemon/internal/config"
)

// TransformChain 文本转换链
// 工作流: 输入文本 -> Prompt模板 -> ChatModel -> 输出转换后的文本
type TransformChain struct {
	chatModel model.BaseChatModel
}

// TransformRequest 转换请求
type TransformRequest struct {
	Text   string // 输入文本
	Prompt string // 转换指令 (如: "翻译成英文", "总结要点", "改写成正式语气")
}

// TransformResponse 转换响应
type TransformResponse struct {
	Text         string // 转换后的文本
	PromptTokens int    // 输入 token 数
	OutputTokens int    // 输出 token 数
}

// NewTransformChain 创建文本转换链
func NewTransformChain(ctx context.Context, cfg *config.AIConfig) (*TransformChain, error) {
	chatModel, err := component.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &TransformChain{
		chatModel: chatModel,
	}, nil
}

// Run 执行文本转换
func (c *TransformChain) Run(ctx context.Context, req *TransformRequest) (*TransformResponse, error) {
	// 构建消息
	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant that transforms text based on user instructions."),
		schema.UserMessage(buildTransformPrompt(req.Prompt, req.Text)),
	}

	// 调用模型
	resp, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 提取 token 使用量
	var promptTokens, outputTokens int
	if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
		promptTokens = resp.ResponseMeta.Usage.PromptTokens
		outputTokens = resp.ResponseMeta.Usage.CompletionTokens
	}

	return &TransformResponse{
		Text:         resp.Content,
		PromptTokens: promptTokens,
		OutputTokens: outputTokens,
	}, nil
}

// Stream 流式执行文本转换
func (c *TransformChain) Stream(ctx context.Context, req *TransformRequest) (*schema.StreamReader[*schema.Message], error) {
	// 构建消息
	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant that transforms text based on user instructions."),
		schema.UserMessage(buildTransformPrompt(req.Prompt, req.Text)),
	}

	// 流式调用模型
	return c.chatModel.Stream(ctx, messages)
}

// buildTransformPrompt 构建转换提示词
func buildTransformPrompt(instruction, text string) string {
	return "Instruction: " + instruction + "\n\nText to transform:\n" + text
}
