package resource

import (
	"lemon/internal/service"
)

// Handler 资源模块处理器
// 所有资源相关的Handler方法都通过这个结构体访问Service
type Handler struct {
	resourceService service.ResourceService
}

// NewHandler 创建资源模块处理器
func NewHandler(resourceService service.ResourceService) *Handler {
	return &Handler{
		resourceService: resourceService,
	}
}
