package ai

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"lemon/internal/config"
	"lemon/internal/model"
)

// Client AI 能力层客户端
// 职责: 封装所有 AI 能力，提供统一接口
type Client struct {
	cfg       *config.AIConfig
	chatChain *ChatChain // 对话链
	// agentGraph *AgentGraph // 智能代理 (后续实现)
}

// NewClient 创建 AI 客户端
func NewClient(cfg *config.AIConfig) (*Client, error) {
	if cfg.APIKey == "" {
		log.Warn().Msg("AI API key not configured, using mock mode")
	}

	// 初始化对话链
	chatChain, err := NewChatChain(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat chain: %w", err)
	}

	return &Client{
		cfg:       cfg,
		chatChain: chatChain,
	}, nil
}

// ChatRequest AI 对话请求
type ChatRequest struct {
	Message string
	History []model.Message
	Options *ChatOptions
}

// ChatResponse AI 对话响应
type ChatResponse struct {
	Content string
	Usage   *model.TokenUsage
}

// ChatOptions AI 对话选项
type ChatOptions struct {
	Temperature float64
	MaxTokens   int
	TopP        float64
}

// Chat 同步对话
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return c.chatChain.Run(ctx, req)
}

// ChatStream 流式对话
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *model.ChatChunk, error) {
	return c.chatChain.Stream(ctx, req)
}

// Close 关闭客户端
func (c *Client) Close() error {
	// 清理资源
	return nil
}
