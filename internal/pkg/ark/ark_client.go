package ark

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"lemon/internal/config"
)

// Client Ark 客户端封装
// 用于调用火山引擎的 Ark API（豆包大模型）
// 使用官方 volcengine-go-sdk
// 参考: https://github.com/volcengine/volcengine-go-sdk
type Client struct {
	client *arkruntime.Client
	model  string
	mu     sync.Mutex // 用于并发安全
}

// ArkConfigFromEnv 从环境变量创建 Ark 配置
// 支持的环境变量：
//   - ARK_API_KEY: API Key（必需）
//   - ARK_MODEL: 模型名称（可选，默认: doubao-seed-1-6-flash-250615）
//   - ARK_BASE_URL: API 基础 URL（可选，默认: https://ark.cn-beijing.volces.com/api/v3）
func ArkConfigFromEnv() *config.AIConfig {
	apiKey := os.Getenv("ARK_API_KEY")
	model := os.Getenv("ARK_MODEL")
	baseURL := os.Getenv("ARK_BASE_URL")

	if model == "" {
		model = "doubao-seed-1-6-flash-250615"
	}
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	return &config.AIConfig{
		Provider: "ark",
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
	}
}

// NewClient 创建 Ark 客户端（使用官方 SDK）
func NewClient(cfg *config.AIConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Ark API key is required")
	}

	// 设置默认值
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	model := cfg.Model
	if model == "" {
		model = "doubao-seed-1-6-flash-250615" // 默认模型
	}

	// 创建客户端选项
	var opts []arkruntime.ConfigOption
	if baseURL != "" {
		opts = append(opts, arkruntime.WithBaseUrl(baseURL))
	}

	// 使用 API Key 创建客户端
	arkClient := arkruntime.NewClientWithApiKey(cfg.APIKey, opts...)

	return &Client{
		client: arkClient,
		model:  model,
	}, nil
}

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model       string    `json:"model"`                 // 模型名称
	Messages    []Message `json:"messages"`              // 消息列表
	MaxTokens   *int      `json:"max_tokens,omitempty"`  // 最大token数
	Temperature *float64  `json:"temperature,omitempty"` // 温度参数
	TopP        *float64  `json:"top_p,omitempty"`       // TopP参数
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`    // user, assistant, system
	Content string `json:"content"` // 消息内容
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice 选择结果
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage Token使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CreateChatCompletion 创建聊天完成（对应 Python 的 client.chat.completions.create）
// 这是主要的 API 调用方法，用于生成文本
func (c *Client) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果没有指定模型，使用客户端默认模型
	if req.Model == "" {
		req.Model = c.model
	}

	// 构建请求参数
	input := &model.ChatCompletionRequest{
		Model:    req.Model,
		Messages: convertMessages(req.Messages),
	}

	if req.MaxTokens != nil {
		input.MaxTokens = *req.MaxTokens
	}

	if req.Temperature != nil {
		input.Temperature = float32(*req.Temperature)
	}

	if req.TopP != nil {
		input.TopP = float32(*req.TopP)
	}

	// 调用 API
	output, err := c.client.CreateChatCompletion(ctx, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to call Ark ChatCompletion API")
		return nil, fmt.Errorf("Ark API call failed: %w", err)
	}

	// 转换响应
	return convertChatCompletionResponse(&output), nil
}

// CreateChatCompletionSimple 简化版本的聊天完成（只需要 prompt）
// 方便快速调用
func (c *Client) CreateChatCompletionSimple(ctx context.Context, prompt string) (string, error) {
	maxTokens := 32 * 1024
	temperature := 0.7

	req := &ChatCompletionRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	resp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// convertMessages 转换消息格式
func convertMessages(messages []Message) []*model.ChatCompletionMessage {
	result := make([]*model.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		content := &model.ChatCompletionMessageContent{
			StringValue: &msg.Content,
		}
		result[i] = &model.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}
	}
	return result
}

// convertChatCompletionResponse 转换响应格式
func convertChatCompletionResponse(output *model.ChatCompletionResponse) *ChatCompletionResponse {
	resp := &ChatCompletionResponse{
		ID:      output.ID,
		Choices: make([]Choice, len(output.Choices)),
	}

	for i, choice := range output.Choices {
		// 提取消息内容
		var content string
		if choice.Message.Content != nil && choice.Message.Content.StringValue != nil {
			content = *choice.Message.Content.StringValue
		}

		resp.Choices[i] = Choice{
			Index: choice.Index,
			Message: Message{
				Role:    choice.Message.Role,
				Content: content,
			},
			FinishReason: string(choice.FinishReason),
		}
	}

	// 转换 Usage
	resp.Usage = &Usage{
		PromptTokens:     output.Usage.PromptTokens,
		CompletionTokens: output.Usage.CompletionTokens,
		TotalTokens:      output.Usage.TotalTokens,
	}

	return resp
}
