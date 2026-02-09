package novel

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
)

// SceneService 场景和镜头服务接口
// 定义场景和镜头相关的能力
type SceneService interface {
	// GetScenesByChapterID 根据章节ID获取场景列表
	GetScenesByChapterID(ctx context.Context, chapterID string) ([]*novel.Scene, error)

	// GetShotsByChapterID 根据章节ID获取镜头列表
	GetShotsByChapterID(ctx context.Context, chapterID string) ([]*novel.Shot, error)

	// UpdateShot 更新分镜头信息
	UpdateShot(ctx context.Context, shotID string, updates map[string]interface{}) error

	// CreateDefaultScenesAndShots 创建默认的场景和镜头（10个场景，每个场景1-4个镜头）
	CreateDefaultScenesAndShots(ctx context.Context, novelID, chapterID, userID string) error

	// SetActiveSceneVersion 设置章节的生效场景版本号
	SetActiveSceneVersion(ctx context.Context, chapterID string, version int) error
}

// GetScenesByChapterID 根据章节ID获取场景列表（默认返回当前生效版本）
func (s *novelService) GetScenesByChapterID(ctx context.Context, chapterID string) ([]*novel.Scene, error) {
	// 获取章节信息，查看当前生效的版本号
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	// 如果有生效版本，查询该版本；否则查询所有版本
	if chapter.ActiveSceneVersion > 0 {
		return s.sceneRepo.FindByChapterIDAndVersion(ctx, chapterID, chapter.ActiveSceneVersion)
	}
	return s.sceneRepo.FindByChapterID(ctx, chapterID)
}

// GetShotsByChapterID 根据章节ID获取镜头列表（默认返回当前生效版本）
func (s *novelService) GetShotsByChapterID(ctx context.Context, chapterID string) ([]*novel.Shot, error) {
	// 获取章节信息，查看当前生效的版本号
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	// 如果有生效版本，查询该版本；否则查询所有版本
	if chapter.ActiveSceneVersion > 0 {
		return s.shotRepo.FindByChapterIDAndVersion(ctx, chapterID, chapter.ActiveSceneVersion)
	}
	return s.shotRepo.FindByChapterID(ctx, chapterID)
}

// UpdateShot 更新分镜头信息
func (s *novelService) UpdateShot(ctx context.Context, shotID string, updates map[string]interface{}) error {
	return s.shotRepo.Update(ctx, shotID, updates)
}

// CreateDefaultScenesAndShots 创建默认的场景和镜头（根据章节内容生成10个场景，每个场景1-4个镜头）
// 每次生成都会创建一个新版本，版本号自动递增
func (s *novelService) CreateDefaultScenesAndShots(ctx context.Context, novelID, chapterID, userID string) error {
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return fmt.Errorf("find chapter: %w", err)
	}

	// 2. 获取章节总数
	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("get chapters: %w", err)
	}
	totalChapters := len(chapters)

	// 3. 获取小说信息（用于获取风格类型）
	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("find novel: %w", err)
	}

	// 4. 计算新版本号（查询当前章节的最大版本号，然后+1）
	newVersion := 1
	existingScenes, err := s.sceneRepo.FindByChapterID(ctx, chapterID)
	if err == nil && len(existingScenes) > 0 {
		maxVersion := 0
		for _, scene := range existingScenes {
			if scene.Version > maxVersion {
				maxVersion = scene.Version
			}
		}
		newVersion = maxVersion + 1
	}

	// 5. 调用 LLM 生成场景和镜头，并解析转换为实体
	generator := noveltools.NewNarrationGenerator(s.llmProvider)
	scenes, shots, characters, props, err := generator.GenerateScenesAndShots(
		ctx,
		chapter.ChapterText,
		chapter.Sequence,
		totalChapters,
		chapter.WordCount,
		chapterID,
		novelID,
		userID,
		novelEntity.Style,
	)
	if err != nil {
		return fmt.Errorf("generate scenes and shots failed: %w", err)
	}

	// 4. 保存角色（如果已存在则更新，不存在则创建）
	now := time.Now()
	for _, char := range characters {
		if char == nil || char.Name == "" {
			continue
		}
		// 检查角色是否已存在
		existing, err := s.characterRepo.FindByNameAndNovelID(ctx, char.Name, novelID)
		if err == nil && existing != nil {
			// 角色已存在，更新信息（保留已有的图片等）
			updates := map[string]interface{}{
				"updated_at": now,
			}
			if char.Gender != "" {
				updates["gender"] = char.Gender
			}
			if char.AgeGroup != "" {
				updates["age_group"] = char.AgeGroup
			}
			if char.RoleNumber != "" {
				updates["role_number"] = char.RoleNumber
			}
			if char.Description != "" {
				updates["description"] = char.Description
			}
			if char.ImagePrompt != "" {
				updates["image_prompt"] = char.ImagePrompt
			}
			if err := s.characterRepo.Update(ctx, existing.ID, updates); err != nil {
				log.Warn().Err(err).
					Str("character_id", existing.ID).
					Str("character_name", existing.Name).
					Msg("更新角色信息失败，继续处理")
			}
		} else {
			// 角色不存在，创建新角色
			char.CreatedAt = now
			char.UpdatedAt = now
			if err := s.characterRepo.Create(ctx, char); err != nil {
				log.Warn().Err(err).
					Str("character_name", char.Name).
					Msg("创建角色失败，继续处理")
			}
		}
	}

	// 5. 保存道具（如果已存在则更新，不存在则创建）
	for _, prop := range props {
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
			if prop.ImagePrompt != "" {
				updates["image_prompt"] = prop.ImagePrompt
			}
			if prop.Category != "" {
				updates["category"] = prop.Category
			}
			if err := s.propRepo.Update(ctx, existing.ID, updates); err != nil {
				log.Warn().Err(err).
					Str("prop_id", existing.ID).
					Str("prop_name", existing.Name).
					Msg("更新道具信息失败，继续处理")
			}
		} else {
			// 道具不存在，创建新道具
			prop.CreatedAt = now
			prop.UpdatedAt = now
			if err := s.propRepo.Create(ctx, prop); err != nil {
				log.Warn().Err(err).
					Str("prop_name", prop.Name).
					Msg("创建道具失败，继续处理")
			}
		}
	}

	// 6. 为场景和镜头生成新的 ID、设置版本号并设置时间戳
	// 创建场景ID映射，用于更新镜头的 SceneID
	sceneIDMap := make(map[string]string) // 旧 sceneID -> 新 sceneID
	for _, scene := range scenes {
		oldSceneID := scene.ID
		scene.ID = id.New()
		sceneIDMap[oldSceneID] = scene.ID
		scene.Version = newVersion // 设置版本号
		scene.CreatedAt = now
		scene.UpdatedAt = now
		if scene.Status == "" {
			scene.Status = novel.TaskStatusPending
		}
	}

	// 更新镜头的 SceneID、生成新的 ID 并设置版本号
	for _, shot := range shots {
		// 更新 SceneID 为新的场景ID
		if newSceneID, ok := sceneIDMap[shot.SceneID]; ok {
			shot.SceneID = newSceneID
		}
		shot.ID = id.New()
		shot.Version = newVersion // 设置版本号（与场景版本号一致）
		shot.CreatedAt = now
		shot.UpdatedAt = now
		if shot.Status == "" {
			shot.Status = novel.TaskStatusPending
		}
	}

	// 7. 保存场景和镜头
	if err := s.sceneRepo.CreateMany(ctx, scenes); err != nil {
		return fmt.Errorf("save scenes failed: %w", err)
	}

	if err := s.shotRepo.CreateMany(ctx, shots); err != nil {
		return fmt.Errorf("save shots failed: %w", err)
	}

	// 8. 自动将新生成的版本设置为生效版本
	// 无论是第一次生成还是重新生成，新版本都应该成为生效版本
	if err := s.chapterRepo.Update(ctx, chapterID, map[string]interface{}{
		"active_scene_version": newVersion,
	}); err != nil {
		log.Warn().Err(err).
			Str("chapter_id", chapterID).
			Int("version", newVersion).
			Msg("设置生效版本失败，继续处理")
	}

	log.Info().
		Str("novel_id", novelID).
		Str("chapter_id", chapterID).
		Int("version", newVersion).
		Int("scenes_count", len(scenes)).
		Int("shots_count", len(shots)).
		Int("characters_count", len(characters)).
		Int("props_count", len(props)).
		Msg("场景、镜头、角色和道具生成完成")

	return nil
}

// SetActiveSceneVersion 设置章节的生效场景版本号
func (s *novelService) SetActiveSceneVersion(ctx context.Context, chapterID string, version int) error {
	// 验证章节是否存在
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return fmt.Errorf("find chapter: %w", err)
	}

	// 验证版本是否存在（检查是否有该版本的场景）
	scenes, err := s.sceneRepo.FindByChapterIDAndVersion(ctx, chapterID, version)
	if err != nil {
		return fmt.Errorf("find scenes by version: %w", err)
	}
	if len(scenes) == 0 {
		return fmt.Errorf("version %d not found for chapter %s", version, chapterID)
	}

	// 更新章节的生效版本号
	if err := s.chapterRepo.Update(ctx, chapterID, map[string]interface{}{
		"active_scene_version": version,
	}); err != nil {
		return fmt.Errorf("update chapter active version: %w", err)
	}

	log.Info().
		Str("chapter_id", chapterID).
		Int("old_version", chapter.ActiveSceneVersion).
		Int("new_version", version).
		Msg("章节生效场景版本号已更新")

	return nil
}
