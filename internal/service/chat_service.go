package service

import (
	"context"

	"github.com/rs/zerolog/log"

	"lemon/internal/ai"
	"lemon/internal/model"
	"lemon/internal/repository"
)

// ChatService 对话服务 - 业务逻辑层
// 职责: 编排 AI 层和数据层，实现业务流程
type ChatService struct {
	aiClient *ai.Client                   // AI 能力层
	convRepo *repository.ConversationRepo // 数据访问层
}

// NewChatService 创建对话服务
func NewChatService(aiClient *ai.Client, convRepo *repository.ConversationRepo) *ChatService {
	return &ChatService{
		aiClient: aiClient,
		convRepo: convRepo,
	}
}

// Chat 处理对话请求
// 业务流程: 1. 获取历史消息 -> 2. 调用 AI -> 3. 保存消息
func (s *ChatService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	logger := log.With().Str("conversation_id", req.ConversationID).Logger()

	// 1. 获取对话历史 (如果有 conversation_id)
	var history []model.Message
	if req.ConversationID != "" && s.convRepo != nil {
		conv, err := s.convRepo.FindByID(ctx, req.ConversationID)
		if err == nil {
			history = conv.Messages
		}
	}

	// 2. 调用 AI 层
	aiReq := &ai.ChatRequest{
		Message: req.Message,
		History: history,
		Options: convertOptions(req.Options),
	}

	aiResp, err := s.aiClient.Chat(ctx, aiReq)
	if err != nil {
		logger.Error().Err(err).Msg("AI chat failed")
		return nil, err
	}

	// 3. 保存消息到数据库 (如果有 conversation_id)
	if req.ConversationID != "" && s.convRepo != nil {
		// 保存用户消息
		userMsg := model.Message{
			Role:    "user",
			Content: req.Message,
		}
		if err := s.convRepo.AppendMessage(ctx, req.ConversationID, userMsg); err != nil {
			logger.Warn().Err(err).Msg("failed to save user message")
		}

		// 保存 AI 回复
		assistantMsg := model.Message{
			Role:       "assistant",
			Content:    aiResp.Content,
			TokenUsage: aiResp.Usage,
		}
		if err := s.convRepo.AppendMessage(ctx, req.ConversationID, assistantMsg); err != nil {
			logger.Warn().Err(err).Msg("failed to save assistant message")
		}
	}

	logger.Info().
		Int("prompt_tokens", aiResp.Usage.PromptTokens).
		Int("completion_tokens", aiResp.Usage.CompletionTokens).
		Msg("chat completed")

	return &model.ChatResponse{
		Message:        aiResp.Content,
		ConversationID: req.ConversationID,
		Usage:          aiResp.Usage,
	}, nil
}

// ChatStream 流式对话
func (s *ChatService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan *model.ChatChunk, error) {
	// 获取历史
	var history []model.Message
	if req.ConversationID != "" && s.convRepo != nil {
		conv, err := s.convRepo.FindByID(ctx, req.ConversationID)
		if err == nil {
			history = conv.Messages
		}
	}

	// 调用 AI 层流式接口
	aiReq := &ai.ChatRequest{
		Message: req.Message,
		History: history,
		Options: convertOptions(req.Options),
	}

	return s.aiClient.ChatStream(ctx, aiReq)
}

// convertOptions 转换选项
func convertOptions(opts *model.ChatOptions) *ai.ChatOptions {
	if opts == nil {
		return nil
	}
	return &ai.ChatOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
		TopP:        opts.TopP,
	}
}
