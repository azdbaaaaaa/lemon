package novel

import (
	"lemon/internal/service/novel"
)

// Handler 小说处理器
// 所有novel相关的Handler方法都通过这个结构体访问Service
type Handler struct {
	novelService novel.NovelService
}

// NewHandler 创建小说处理器
func NewHandler(novelService novel.NovelService) *Handler {
	return &Handler{
		novelService: novelService,
	}
}
