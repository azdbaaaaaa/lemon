package model

// ChatResponse 对话响应
type ChatResponse struct {
	Message        string      `json:"message"`
	ConversationID string      `json:"conversation_id,omitempty"`
	Usage          *TokenUsage `json:"usage,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatChunk 流式对话片段
type ChatChunk struct {
	Content string      `json:"content,omitempty"`
	Done    bool        `json:"done,omitempty"`
	Usage   *TokenUsage `json:"usage,omitempty"`
}

// TransformResponse 文本转换响应
type TransformResponse struct {
	Text  string      `json:"text"`            // 转换后的文本
	Usage *TokenUsage `json:"usage,omitempty"` // Token 使用统计
}
