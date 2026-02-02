package novel

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

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
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return fmt.Errorf("find chapter narration: %w", err)
	}

	if narration.Content == nil || narration.Content.Characters == nil {
		return nil
	}

	for _, narrationChar := range narration.Content.Characters {
		// 检查是否已存在同名角色
		existing, err := s.characterRepo.FindByNameAndNovelID(ctx, narrationChar.Name, novelID)
		if err != nil && err.Error() != "mongo: no documents in result" {
			// 忽略"未找到"错误，继续创建新角色
			continue
		}

		if existing != nil {
			// 更新现有角色信息（合并外貌特征等）
			updates := bson.M{}
			if narrationChar.Gender != "" {
				updates["gender"] = narrationChar.Gender
			}
			if narrationChar.AgeGroup != "" {
				updates["age_group"] = narrationChar.AgeGroup
			}
			if narrationChar.RoleNumber != "" {
				updates["role_number"] = narrationChar.RoleNumber
			}

			// 注意：这里简化处理，实际应该从 NarrationCharacter 中提取更详细的外貌和服装信息
			// 但由于 NarrationCharacter 结构较简单，这里只更新基本信息
			if err := s.characterRepo.Update(ctx, existing.ID, updates); err != nil {
				return fmt.Errorf("update character %s: %w", existing.ID, err)
			}
		} else {
			// 创建新角色
			character := &novel.Character{
				ID:         id.New(),
				NovelID:    novelID,
				Name:       narrationChar.Name,
				Gender:     narrationChar.Gender,
				AgeGroup:   narrationChar.AgeGroup,
				RoleNumber: narrationChar.RoleNumber,
				// 注意：NarrationCharacter 中没有详细的外貌和服装信息
				// 这些信息需要从其他来源获取，或者后续通过其他方式补充
			}

			if err := s.characterRepo.Create(ctx, character); err != nil {
				return fmt.Errorf("create character %s: %w", character.Name, err)
			}
		}
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
