package novel

import (
	"context"

	"lemon/internal/model/novel"
)

// PropService 道具服务接口
// 定义道具相关的能力
type PropService interface {
	// GetPropsByNovelID 获取小说的所有道具
	GetPropsByNovelID(ctx context.Context, novelID string) ([]*novel.Prop, error)
}

// GetPropsByNovelID 获取小说的所有道具
func (s *novelService) GetPropsByNovelID(ctx context.Context, novelID string) ([]*novel.Prop, error) {
	return s.propRepo.FindByNovelID(ctx, novelID)
}
