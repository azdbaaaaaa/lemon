package http

// ErrorResponse 错误响应（所有API共用）
// 用于统一错误响应格式
type ErrorResponse struct {
	Code    int    `json:"code"`             // 错误码（非0表示错误）
	Message string `json:"message"`          // 错误消息
	Detail  string `json:"detail,omitempty"` // 错误详情（可选）
}

// SuccessResponse 成功响应（所有API共用）
// 用于统一成功响应格式
type SuccessResponse struct {
	Code    int         `json:"code"`           // 状态码（0表示成功）
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据（可选）
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Code:    0,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string, detail ...string) *ErrorResponse {
	resp := &ErrorResponse{
		Code:    code,
		Message: message,
	}
	if len(detail) > 0 && detail[0] != "" {
		resp.Detail = detail[0]
	}
	return resp
}
