package service

import (
	"context"

	"github.com/rs/zerolog/log"

	"lemon/internal/ai/chain"
	"lemon/internal/config"
	"lemon/internal/model"
)

// TransformService 文本转换服务
type TransformService struct {
	transformChain *chain.TransformChain
}

// NewTransformService 创建文本转换服务
func NewTransformService(ctx context.Context, cfg *config.AIConfig) (*TransformService, error) {
	transformChain, err := chain.NewTransformChain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &TransformService{
		transformChain: transformChain,
	}, nil
}

// Transform 执行文本转换
func (s *TransformService) Transform(ctx context.Context, req *model.TransformRequest) (*model.TransformResponse, error) {
	logger := log.With().Str("prompt", req.Prompt).Logger()

	// 调用转换链
	chainReq := &chain.TransformRequest{
		Text:   req.Text,
		Prompt: req.Prompt,
	}

	resp, err := s.transformChain.Run(ctx, chainReq)
	if err != nil {
		logger.Error().Err(err).Msg("transform failed")
		return nil, err
	}

	logger.Info().
		Int("prompt_tokens", resp.PromptTokens).
		Int("output_tokens", resp.OutputTokens).
		Msg("transform completed")

	return &model.TransformResponse{
		Text: resp.Text,
		Usage: &model.TokenUsage{
			PromptTokens:     resp.PromptTokens,
			CompletionTokens: resp.OutputTokens,
			TotalTokens:      resp.PromptTokens + resp.OutputTokens,
		},
	}, nil
}
