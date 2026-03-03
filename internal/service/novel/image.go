package novel

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/service"

	"github.com/rs/zerolog/log"
)

// ImageService 章节图片服务接口
// 定义章节图片相关的能力
type ImageService interface {
	// GenerateImagesForNarration 为章节解说生成所有章节图片
	// 自动使用最新的版本号+1
	GenerateImagesForNarration(ctx context.Context, narrationID string) ([]string, error)

	// GenerateCharacterImages 为小说的所有角色生成图片（异步执行，立即返回）
	GenerateCharacterImages(ctx context.Context, novelID string) error

	// GenerateSceneImages 为解说的所有场景生成图片
	GenerateSceneImages(ctx context.Context, narrationID string) ([]string, error)

	// GeneratePropImages 为小说的所有道具生成图片（异步执行，立即返回）
	GeneratePropImages(ctx context.Context, novelID string) error

	// GetImageVersions 获取章节的所有图片版本号
	GetImageVersions(ctx context.Context, chapterID string) ([]int, error)

	// ListImagesByNarration 获取解说的图片列表（可指定版本；version<=0 则取最新版本）
	ListImagesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Image, int, error)

	// GetCharacterImages 获取角色的所有图片
	GetCharacterImages(ctx context.Context, characterID string) ([]*novel.Image, error)

	// GetPropImages 获取道具的所有图片
	GetPropImages(ctx context.Context, propID string) ([]*novel.Image, error)

	// GenerateShotImages 为单个镜头生成首图和尾图
	GenerateShotImages(ctx context.Context, shotID string) ([]string, error)

	// GetShotImages 获取镜头的所有图片（首图和尾图）
	GetShotImages(ctx context.Context, shotID string) ([]*novel.Image, error)
}

// GenerateImagesForNarration 为章节解说生成所有章节图片
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GenerateImagesForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// GetImageVersions 获取章节的所有图片版本号
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GetImageVersions(ctx context.Context, chapterID string) ([]int, error) {
	// TODO: 重构此方法，根据章节ID查询图片版本号
	// 暂时返回空列表
	return []int{1}, nil
}

// getNextImageVersion 获取章节的下一个图片版本号（自动递增）
// chapterID: 章节ID
// baseVersion: 基础版本号（如 1），如果为0则自动生成下一个版本号
func (s *novelService) getNextImageVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	// TODO: 重构此方法，根据 shot_id 或 chapter_id 查询图片版本号
	// 暂时返回基础版本号或1
	if baseVersion == 0 {
		return 1, nil
	}
	return baseVersion, nil
}

// GenerateCharacterImages 为小说的所有角色生成图片（异步执行，立即返回）
// 为每个角色生成三种细分类的图片：正视图、三视图、细节图
func (s *novelService) GenerateCharacterImages(ctx context.Context, novelID string) error {
	// 检查小说是否存在
	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("find novel: %w", err)
	}

	// 获取所有角色
	characters, err := s.characterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("find characters: %w", err)
	}

	if len(characters) == 0 {
		return nil
	}

	// 将所有角色状态设置为 pending
	for _, char := range characters {
		if err := s.characterRepo.UpdateStatus(ctx, char.ID, novel.TaskStatusPending, ""); err != nil {
			log.Warn().Err(err).Str("character_id", char.ID).Msg("更新角色状态失败")
		}
	}

	// 启动异步任务
	go s.generateCharacterImagesAsync(context.Background(), novelEntity, characters)

	return nil
}

// generateCharacterImagesAsync 异步生成角色图片
func (s *novelService) generateCharacterImagesAsync(ctx context.Context, novelEntity *novel.Novel, characters []*novel.Character) {
	// 定义三种细分类
	subtypes := []novel.CharacterImageSubtype{
		novel.CharacterImageSubtypeFront,
		novel.CharacterImageSubtypeThreeView,
		novel.CharacterImageSubtypeDetail,
	}

	for _, char := range characters {
		// 为每个角色生成三种细分类的图片
		hasError := false
		var errorMessages []string

		for _, subtype := range subtypes {
			// 检查该细分类是否已有图片
			existingImages, err := s.imageRepo.FindByCharacterIDAndSubtype(ctx, char.ID, subtype)
			if err == nil && len(existingImages) > 0 {
				log.Info().
					Str("character_id", char.ID).
					Str("character_name", char.Name).
					Str("subtype", string(subtype)).
					Int("count", len(existingImages)).
					Msg("角色图片已存在，跳过")
				continue
			}

			imageID, err := s.generateCharacterImage(ctx, novelEntity, char, subtype)
			if err != nil {
				log.Error().
					Err(err).
					Str("character_id", char.ID).
					Str("character_name", char.Name).
					Str("subtype", string(subtype)).
					Msg("生成角色图片失败")
				hasError = true
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %v", subtype, err))
			} else {
				log.Info().
					Str("character_id", char.ID).
					Str("character_name", char.Name).
					Str("subtype", string(subtype)).
					Str("image_id", imageID).
					Msg("角色图片生成成功")
			}
		}

		// 更新角色状态
		if hasError {
			errorMsg := strings.Join(errorMessages, "; ")
			if err := s.characterRepo.UpdateStatus(ctx, char.ID, novel.TaskStatusFailed, errorMsg); err != nil {
				log.Warn().Err(err).Str("character_id", char.ID).Msg("更新角色状态失败")
			}
		} else {
			if err := s.characterRepo.UpdateStatus(ctx, char.ID, novel.TaskStatusCompleted, ""); err != nil {
				log.Warn().Err(err).Str("character_id", char.ID).Msg("更新角色状态失败")
			}
		}
	}
}

// buildCharacterImagePrompt 基于角色结构化信息和 Prompt 模块生成角色图片提示词
func (s *novelService) buildCharacterImagePrompt(ctx context.Context, char *novel.Character) (string, error) {
	if s.promptService == nil {
		return "", fmt.Errorf("promptService is not initialized")
	}

	base := char.BaseProfile
	var appearance *novel.CharacterAppearance
	var clothing *novel.CharacterClothing
	var sig *novel.CharacterSignatureElements

	if base != nil {
		appearance = base.Appearance
		clothing = base.Clothing
		sig = base.SignatureElements
	}

	// 兼容老字段：如果 BaseProfile 为空，则尽量从旧字段补充
	gender := char.Gender
	ageRange := char.AgeGroup
	bodyType := ""
	if base != nil {
		if base.Gender != "" {
			gender = base.Gender
		}
		if base.AgeRange != "" {
			ageRange = base.AgeRange
		}
		bodyType = base.BodyType
	}

	var temperamentKeywords string
	if base != nil && len(base.VisualPersonalityKeywords) > 0 {
		temperamentKeywords = strings.Join(base.VisualPersonalityKeywords, ", ")
	}

	var signatureItem string
	if sig != nil {
		if sig.Items != "" {
			signatureItem = sig.Items
		} else if sig.Weapon != "" {
			signatureItem = sig.Weapon
		} else if sig.Accessories != "" {
			signatureItem = sig.Accessories
		}
	}

	vars := map[string]string{
		"CHARACTER_NAME":       char.Name,
		"GENDER":               gender,
		"AGE_RANGE":            ageRange,
		"BODY_TYPE":            bodyType,
		"FACE_SHAPE":           firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.FaceShape }),
		"FACIAL_FEATURES":      firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.FacialFeatures }),
		"EYE_COLOR":            firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.EyeColor }),
		"HAIR_STYLE":           firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.HairStyle }),
		"HAIR_COLOR":           firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.HairColor }),
		"SKIN_TONE":            firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.SkinTone }),
		"SPECIAL_MARKS":        firstNonEmpty(appearance, func(a *novel.CharacterAppearance) string { return a.SpecialMarks }),
		"CLOTHING_TYPE":        firstNonEmpty(clothing, func(c *novel.CharacterClothing) string { return c.CommonType }),
		"COLOR_PALETTE":        firstNonEmpty(clothing, func(c *novel.CharacterClothing) string { return c.ColorPalette }),
		"MATERIAL_STYLE":       firstNonEmpty(clothing, func(c *novel.CharacterClothing) string { return c.MaterialStyle }),
		"ERA_SETTING":          firstNonEmpty(clothing, func(c *novel.CharacterClothing) string { return c.EraSetting }),
		"SIGNATURE_ITEM":       signatureItem,
		"TEMPERAMENT_KEYWORDS": temperamentKeywords,
	}

	return s.promptService.RenderPrompt(ctx, "character_image", "default", "en-US", vars)
}

func firstNonEmpty[T any](ptr *T, getter func(*T) string) string {
	if ptr == nil {
		return ""
	}
	return getter(ptr)
}

// generateCharacterImage 生成单个角色图片
func (s *novelService) generateCharacterImage(ctx context.Context, novelEntity *novel.Novel, char *novel.Character, subtype novel.CharacterImageSubtype) (string, error) {
	// 基于角色结构化信息和 Prompt 模块生成基础图片提示词
	basePrompt, err := s.buildCharacterImagePrompt(ctx, char)
	if err != nil {
		return "", fmt.Errorf("build character image prompt: %w", err)
	}

	prompt := basePrompt
	switch subtype {
	case novel.CharacterImageSubtypeFront:
		prompt = prompt + "，正视图"
	case novel.CharacterImageSubtypeThreeView:
		prompt = prompt + "，三视图（正面、侧面、背面）"
	case novel.CharacterImageSubtypeDetail:
		prompt = prompt + "，细节图（特写）"
	}

	// 添加角色图片生成要求：只生成角色人物图，不包含背景、其他人物、其他物品等
	prompt = prompt + "，纯色背景或透明背景，只包含该角色人物，不包含其他人物、物品、道具、场景背景等，人物全身或半身，高质量角色立绘"

	outputFilename := fmt.Sprintf("character_%s_%s.jpeg", char.Name, subtype)

	imageData, err := s.imageProvider.GenerateImage(ctx, prompt, outputFilename)
	if err != nil {
		return "", fmt.Errorf("generate image: %w", err)
	}

	uploadReq := &service.UploadFileRequest{
		UserID:      novelEntity.UserID,
		FileName:    outputFilename,
		ContentType: "image/jpeg",
		Ext:         "jpeg",
		Data:        bytes.NewReader(imageData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}

	// 将图片保存到 Image 表
	imageID := id.New()
	characterImage := &novel.Image{
		ID:                    imageID,
		NovelID:               novelEntity.ID,
		ImageType:             novel.ImageTypeCharacter,
		CharacterID:           char.ID,
		CharacterImageSubtype: subtype,
		ImageResourceID:       uploadResult.ResourceID,
		Prompt:                prompt,
		Version:               1,
		Status:                novel.TaskStatusCompleted,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.imageRepo.Create(ctx, characterImage); err != nil {
		return "", fmt.Errorf("create character image record: %w", err)
	}

	log.Info().
		Str("character_id", char.ID).
		Str("character_name", char.Name).
		Str("subtype", string(subtype)).
		Str("image_id", imageID).
		Msg("角色图片生成成功")
	return imageID, nil
}

// GenerateSceneImages 为解说的所有场景生成图片
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GenerateSceneImages(ctx context.Context, narrationID string) ([]string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// GeneratePropImages 为小说的所有道具生成图片（异步执行，立即返回）
func (s *novelService) GeneratePropImages(ctx context.Context, novelID string) error {
	// 检查小说是否存在
	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("find novel: %w", err)
	}

	// 获取所有道具
	props, err := s.propRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return fmt.Errorf("find props: %w", err)
	}

	if len(props) == 0 {
		return nil
	}

	// 将所有道具状态设置为 pending
	for _, prop := range props {
		if err := s.propRepo.UpdateStatus(ctx, prop.ID, novel.TaskStatusPending, ""); err != nil {
			log.Warn().Err(err).Str("prop_id", prop.ID).Msg("更新道具状态失败")
		}
	}

	// 启动异步任务
	go s.generatePropImagesAsync(context.Background(), novelEntity, props)

	return nil
}

// generatePropImagesAsync 异步生成道具图片
func (s *novelService) generatePropImagesAsync(ctx context.Context, novelEntity *novel.Novel, props []*novel.Prop) {
	for _, prop := range props {
		if prop.ImagePrompt == "" {
			log.Warn().Str("prop_id", prop.ID).Str("prop_name", prop.Name).Msg("道具图片提示词为空，跳过")
			// 更新状态为失败
			if err := s.propRepo.UpdateStatus(ctx, prop.ID, novel.TaskStatusFailed, "图片提示词为空"); err != nil {
				log.Warn().Err(err).Str("prop_id", prop.ID).Msg("更新道具状态失败")
			}
			continue
		}

		// 检查道具是否已有图片（通过查询 Image 表）
		existingImages, err := s.imageRepo.FindByPropID(ctx, prop.ID)
		if err == nil && len(existingImages) > 0 {
			log.Info().Str("prop_id", prop.ID).Str("prop_name", prop.Name).Int("count", len(existingImages)).Msg("道具图片已存在，跳过")
			// 更新状态为已完成
			if err := s.propRepo.UpdateStatus(ctx, prop.ID, novel.TaskStatusCompleted, ""); err != nil {
				log.Warn().Err(err).Str("prop_id", prop.ID).Msg("更新道具状态失败")
			}
			continue
		}

		imageID, err := s.generatePropImage(ctx, novelEntity, prop)
		if err != nil {
			log.Error().Err(err).Str("prop_id", prop.ID).Str("prop_name", prop.Name).Msg("生成道具图片失败")
			// 更新状态为失败
			if err := s.propRepo.UpdateStatus(ctx, prop.ID, novel.TaskStatusFailed, err.Error()); err != nil {
				log.Warn().Err(err).Str("prop_id", prop.ID).Msg("更新道具状态失败")
			}
		} else {
			log.Info().Str("prop_id", prop.ID).Str("prop_name", prop.Name).Str("image_id", imageID).Msg("道具图片生成成功")
			// 更新状态为已完成
			if err := s.propRepo.UpdateStatus(ctx, prop.ID, novel.TaskStatusCompleted, ""); err != nil {
				log.Warn().Err(err).Str("prop_id", prop.ID).Msg("更新道具状态失败")
			}
		}
	}
}

// generatePropImage 生成单个道具图片
func (s *novelService) generatePropImage(ctx context.Context, novelEntity *novel.Novel, prop *novel.Prop) (string, error) {
	// 添加道具图片生成要求：纯白背景，只包含道具本身，不包含其他元素
	prompt := prop.ImagePrompt + "，纯白色背景，只包含该道具物品，不包含其他人物、物品、道具、场景背景等，高质量道具展示图"

	outputFilename := fmt.Sprintf("prop_%s.jpeg", prop.Name)

	imageData, err := s.imageProvider.GenerateImage(ctx, prompt, outputFilename)
	if err != nil {
		return "", fmt.Errorf("generate image: %w", err)
	}

	uploadReq := &service.UploadFileRequest{
		UserID:      novelEntity.UserID,
		FileName:    outputFilename,
		ContentType: "image/jpeg",
		Ext:         "jpeg",
		Data:        bytes.NewReader(imageData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}

	// 道具图片保存到 Image 表
	imageID := id.New()
	propImage := &novel.Image{
		ID:              imageID,
		NovelID:         novelEntity.ID,
		PropID:          prop.ID,
		ImageType:       novel.ImageTypeProp,
		ImageResourceID: uploadResult.ResourceID,
		Prompt:          prompt,
		Version:         1,
		Status:          novel.TaskStatusCompleted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.imageRepo.Create(ctx, propImage); err != nil {
		return "", fmt.Errorf("create prop image record: %w", err)
	}

	log.Info().Str("prop_id", prop.ID).Str("prop_name", prop.Name).Str("image_id", imageID).Msg("道具图片生成成功")
	return imageID, nil
}

// GetCharacterImages 获取角色的所有图片
func (s *novelService) GetCharacterImages(ctx context.Context, characterID string) ([]*novel.Image, error) {
	return s.imageRepo.FindByCharacterID(ctx, characterID)
}

// GetPropImages 获取道具的所有图片
func (s *novelService) GetPropImages(ctx context.Context, propID string) ([]*novel.Image, error) {
	return s.imageRepo.FindByPropID(ctx, propID)
}

// GenerateShotImages 为单个镜头生成首图和尾图
func (s *novelService) GenerateShotImages(ctx context.Context, shotID string) ([]string, error) {
	// 1. 获取镜头信息
	shot, err := s.shotRepo.FindByID(ctx, shotID)
	if err != nil {
		return nil, fmt.Errorf("find shot: %w", err)
	}

	// 2. 检查提示词
	if shot.FirstImagePrompt == "" && shot.LastImagePrompt == "" {
		return nil, fmt.Errorf("shot has no image prompts")
	}

	// 3. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, shot.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	// 4. 获取下一个图片版本号
	nextVersion, err := s.getNextImageVersion(ctx, shot.ChapterID, shot.Version)
	if err != nil {
		return nil, fmt.Errorf("get next image version: %w", err)
	}

	var imageIDs []string

	// 5. 生成首图
	if shot.FirstImagePrompt != "" {
		imageID, err := s.generateShotImage(ctx, shot, chapter, novel.ImageTypeShotFirst, shot.FirstImagePrompt, nextVersion)
		if err != nil {
			log.Error().Err(err).Str("shot_id", shotID).Msg("生成首图失败")
		} else {
			imageIDs = append(imageIDs, imageID)
		}
	}

	// 6. 生成尾图
	if shot.LastImagePrompt != "" {
		imageID, err := s.generateShotImage(ctx, shot, chapter, novel.ImageTypeShotLast, shot.LastImagePrompt, nextVersion)
		if err != nil {
			log.Error().Err(err).Str("shot_id", shotID).Msg("生成尾图失败")
		} else {
			imageIDs = append(imageIDs, imageID)
		}
	}

	return imageIDs, nil
}

// generateShotImage 生成单个镜头图片（首图或尾图）
func (s *novelService) generateShotImage(ctx context.Context, shot *novel.Shot, chapter *novel.Chapter, imageType novel.ImageType, prompt string, version int) (string, error) {
	// 1. 生成图片
	outputFilename := fmt.Sprintf("shot_%s_%s_%s.jpeg", shot.ID, imageType, id.New())
	imageData, err := s.imageProvider.GenerateImage(ctx, prompt, outputFilename)
	if err != nil {
		return "", fmt.Errorf("generate image: %w", err)
	}

	// 2. 上传图片
	uploadReq := &service.UploadFileRequest{
		UserID:      chapter.UserID,
		FileName:    outputFilename,
		ContentType: "image/jpeg",
		Ext:         "jpeg",
		Data:        bytes.NewReader(imageData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}

	// 3. 保存图片记录
	imageID := id.New()
	shotImage := &novel.Image{
		ID:              imageID,
		NovelID:         chapter.NovelID,
		ImageType:       imageType,
		ShotID:          shot.ID,
		ImageResourceID: uploadResult.ResourceID,
		Prompt:          prompt,
		Version:         version,
		Status:          novel.TaskStatusCompleted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.imageRepo.Create(ctx, shotImage); err != nil {
		return "", fmt.Errorf("create shot image record: %w", err)
	}

	log.Info().
		Str("shot_id", shot.ID).
		Str("image_type", string(imageType)).
		Str("image_id", imageID).
		Msg("镜头图片生成成功")

	return imageID, nil
}

// GetShotImages 获取镜头的所有图片（首图和尾图）
func (s *novelService) GetShotImages(ctx context.Context, shotID string) ([]*novel.Image, error) {
	// 查询首图和尾图（version=0 表示查询所有版本）
	firstImages, err := s.imageRepo.FindByShotIDAndTypeAndVersion(ctx, shotID, novel.ImageTypeShotFirst, 0)
	if err != nil {
		return nil, fmt.Errorf("find first images: %w", err)
	}

	lastImages, err := s.imageRepo.FindByShotIDAndTypeAndVersion(ctx, shotID, novel.ImageTypeShotLast, 0)
	if err != nil {
		return nil, fmt.Errorf("find last images: %w", err)
	}

	// 合并结果
	var allImages []*novel.Image
	allImages = append(allImages, firstImages...)
	allImages = append(allImages, lastImages...)

	return allImages, nil
}
