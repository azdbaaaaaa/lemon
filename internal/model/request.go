package model

// ChatRequest 对话请求
type ChatRequest struct {
	Message        string       `json:"message" binding:"required"`
	ConversationID string       `json:"conversation_id,omitempty"`
	Options        *ChatOptions `json:"options,omitempty"`
}

// ChatOptions 对话选项
type ChatOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// CreateConversationRequest 创建对话请求
type CreateConversationRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Title  string `json:"title,omitempty"`
	Model  string `json:"model,omitempty"`
}

// TransformRequest 文本转换请求
type TransformRequest struct {
	Text   string `json:"text" binding:"required"`   // 输入文本
	Prompt string `json:"prompt" binding:"required"` // 转换指令 (如: "翻译成英文", "总结要点")
}
