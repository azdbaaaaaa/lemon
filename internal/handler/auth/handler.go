package auth

import (
	"lemon/internal/service"
)

// Handler 认证处理器
// 所有auth相关的Handler方法都通过这个结构体访问Service
type Handler struct {
	authService *service.AuthService
}

// NewHandler 创建认证处理器
func NewHandler(authService *service.AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}
