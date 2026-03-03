package novel

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

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

func (s *novelService) GeneratePropsFromNovel(ctx context.Context, novelID string) error {
	jsonContent, err := s.generateCharactersAndPropsJSON(ctx, novelID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, prop := range jsonContent.Props {
		if prop == nil || prop.Name == "" {
			continue
		}
		// 检查道具是否已存在
		existing, err := s.propRepo.FindByName(ctx, novelID, prop.Name)
		if err == nil && existing != nil {
			// 道具已存在，更新信息（保留已有的图片等）
			updates := map[string]interface{}{
				"updated_at": now,
			}
			if prop.Description != "" {
				updates["description"] = prop.Description
			}
			if prop.Category != "" {
				updates["category"] = prop.Category
			}
			if prop.ImagePrompt != "" {
				updates["image_prompt"] = prop.ImagePrompt
			}
			if err := s.propRepo.Update(ctx, existing.ID, updates); err != nil {
				log.Warn().Err(err).
					Str("prop_id", existing.ID).
					Str("prop_name", existing.Name).
					Msg("更新道具信息失败，继续处理")
			}
		} else {
			// 道具不存在，创建新道具
			prop.NovelID = novelID
			prop.CreatedAt = now
			prop.UpdatedAt = now
			if err := s.propRepo.Create(ctx, prop); err != nil {
				log.Warn().Err(err).
					Str("prop_name", prop.Name).
					Msg("创建道具失败，继续处理")
			}
		}
	}

	log.Info().
		Str("novel_id", novelID).
		Int("props_count", len(jsonContent.Props)).
		Msg("从小说内容生成道具成功")

	return nil
}
