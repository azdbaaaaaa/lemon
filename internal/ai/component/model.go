package component

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"

	"lemon/internal/config"
)

// NewChatModel 创建 ChatModel
// 支持多种 Provider: openai, azure, anthropic
func NewChatModel(ctx context.Context, cfg *config.AIConfig) (model.ChatModel, error) {
	switch cfg.Provider {
	case "openai", "":
		return newOpenAIChatModel(ctx, cfg)
	case "azure":
		return newAzureChatModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

// newOpenAIChatModel 创建 OpenAI ChatModel
func newOpenAIChatModel(ctx context.Context, cfg *config.AIConfig) (model.ChatModel, error) {
	modelCfg := &openai.ChatModelConfig{
		Model:  cfg.Model,
		APIKey: cfg.APIKey,
	}

	// Base URL (用于代理或兼容 API)
	if cfg.BaseURL != "" {
		modelCfg.BaseURL = cfg.BaseURL
	}

	// 模型参数
	if cfg.Options.Temperature > 0 {
		temp := float32(cfg.Options.Temperature)
		modelCfg.Temperature = &temp
	}
	if cfg.Options.MaxTokens > 0 {
		modelCfg.MaxTokens = &cfg.Options.MaxTokens
	}
	if cfg.Options.TopP > 0 {
		topP := float32(cfg.Options.TopP)
		modelCfg.TopP = &topP
	}

	return openai.NewChatModel(ctx, modelCfg)
}

// newAzureChatModel 创建 Azure OpenAI ChatModel
func newAzureChatModel(ctx context.Context, cfg *config.AIConfig) (model.ChatModel, error) {
	modelCfg := &openai.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		ByAzure: true,
	}

	if cfg.Options.Temperature > 0 {
		temp := float32(cfg.Options.Temperature)
		modelCfg.Temperature = &temp
	}

	return openai.NewChatModel(ctx, modelCfg)
}
