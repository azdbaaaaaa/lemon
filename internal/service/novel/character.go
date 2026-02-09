package novel

import (
	"context"

	"lemon/internal/model/novel"
)

// CharacterService 角色服务接口
// 定义角色相关的能力
type CharacterService interface {
	// GetCharactersByNovelID 获取小说的所有角色
	GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error)

	// GetCharacterByName 根据名称获取角色
	GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error)
}

// GetCharactersByNovelID 获取小说的所有角色
func (s *novelService) GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error) {
	return s.characterRepo.FindByNovelID(ctx, novelID)
}

// GetCharacterByName 根据名称获取角色
func (s *novelService) GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error) {
	return s.characterRepo.FindByNameAndNovelID(ctx, name, novelID)
}
