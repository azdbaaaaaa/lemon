package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"lemon/internal/model"
)

// ChatHandler 对话处理器
type ChatHandler struct{}

// NewChatHandler 创建对话处理器
func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

// Chat 对话接口
func (h *ChatHandler) Chat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// TODO: 集成 Eino 实现真正的 AI 对话
	// 当前返回模拟响应
	resp := model.ChatResponse{
		Message:        "Hello! This is a placeholder response. Eino integration coming soon.",
		ConversationID: req.ConversationID,
		Usage: &model.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	c.JSON(http.StatusOK, resp)
}

// ChatStream 流式对话接口 (SSE)
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// TODO: 集成 Eino 实现真正的流式响应
	// 当前返回模拟响应
	c.Stream(func(w io.Writer) bool {
		c.SSEvent("message", gin.H{"content": "Hello! "})
		c.SSEvent("message", gin.H{"content": "This is a streaming response. "})
		c.SSEvent("message", gin.H{"content": "Eino integration coming soon."})
		c.SSEvent("done", gin.H{
			"usage": model.TokenUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		})
		return false
	})
}
