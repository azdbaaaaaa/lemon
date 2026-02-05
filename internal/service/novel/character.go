package novel

import (
	"context"
	"fmt"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
)

// CharacterService 角色服务接口
// 定义角色相关的能力
type CharacterService interface {
	// SyncCharactersFromNarration 从章节解说同步角色信息到小说级别
	SyncCharactersFromNarration(ctx context.Context, novelID, narrationID string) error

	// GetCharactersByNovelID 获取小说的所有角色
	GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error)

	// GetCharacterByName 根据名称获取角色
	GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error)
}

// SyncCharactersFromNarration 从章节解说同步角色信息到小说级别
func (s *novelService) SyncCharactersFromNarration(ctx context.Context, novelID, narrationID string) error {
	// 获取小说信息以获取 workflow_id
	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("find novel: %w", err)
	}

	// 从独立的表中查询所有镜头，提取角色信息
	shots, err := s.shotRepo.FindByNarrationID(ctx, narrationID)
	if err != nil {
		return fmt.Errorf("find shots: %w", err)
	}

	// 收集所有唯一的角色名称
	characterMap := make(map[string]bool)
	for _, shot := range shots {
		if shot.Character != "" {
			characterMap[shot.Character] = true
		}
	}

	// 为每个角色创建或更新角色记录
	for characterName := range characterMap {
		// 检查是否已存在同名角色
		existing, err := s.characterRepo.FindByNameAndNovelID(ctx, characterName, novelID)
		if err != nil && err.Error() != "mongo: no documents in result" {
			// 忽略"未找到"错误，继续创建新角色
			continue
		}

		if existing == nil {
			// 创建新角色（基本信息，详细外貌信息需要后续补充）
			character := &novel.Character{
				ID:          id.New(),
				NovelID:     novelID,
				WorkflowID:  novelEntity.WorkflowID,
				Name:        characterName,
				// 注意：从 Shot 表中只能获取角色名称，其他信息（性别、年龄等）需要后续补充
			}

			if err := s.characterRepo.Create(ctx, character); err != nil {
				return fmt.Errorf("create character %s: %w", character.Name, err)
			}
		}
		// 如果角色已存在，不需要更新（因为 Shot 表中只有角色名称，没有其他信息）
	}

	return nil
}

// GetCharactersByNovelID 获取小说的所有角色
func (s *novelService) GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error) {
	return s.characterRepo.FindByNovelID(ctx, novelID)
}

// GetCharacterByName 根据名称获取角色
func (s *novelService) GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error) {
	return s.characterRepo.FindByNameAndNovelID(ctx, name, novelID)
}
