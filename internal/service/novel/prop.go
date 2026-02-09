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

	// GeneratePropsFromNovel 基于整个小说内容生成道具和图片提示词
	GeneratePropsFromNovel(ctx context.Context, novelID string) error
}

// GetPropsByNovelID 获取小说的所有道具
func (s *novelService) GetPropsByNovelID(ctx context.Context, novelID string) ([]*novel.Prop, error) {
	return s.propRepo.FindByNovelID(ctx, novelID)
}

// GeneratePropsFromNovel 基于整个小说内容生成道具和图片提示词
// 注意：此方法与 GenerateCharactersFromNovel 一起调用，因为它们使用相同的 prompt
func (s *novelService) GeneratePropsFromNovel(ctx context.Context, novelID string) error {
	// 道具生成逻辑已经在 GenerateCharactersFromNovel 中实现
	// 这里直接调用 GenerateCharactersFromNovel，因为它会同时生成角色和道具
	return s.GenerateCharactersFromNovel(ctx, novelID)
}
