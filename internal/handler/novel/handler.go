package novel

import (
	novelservice "lemon/internal/service/novel"
	"lemon/internal/service"
)

// Handler 内容处理器
// 所有内容相关的Handler方法都通过这个结构体访问Service
type Handler struct {
	novelService    novelservice.NovelService
	contentService  novelservice.ContentService
	resourceService service.ResourceService
}

// NewHandler 创建内容处理器
func NewHandler(novelService novelservice.NovelService, contentService novelservice.ContentService, resourceService service.ResourceService) *Handler {
	return &Handler{
		novelService:    novelService,
		contentService:  contentService,
		resourceService: resourceService,
	}
}
