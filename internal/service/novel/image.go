package novel

import (
	"bytes"
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/service"
)

// ImageService 章节图片服务接口
// 定义章节图片相关的能力
type ImageService interface {
	// GenerateImagesForNarration 为章节解说生成所有章节图片
	// 自动使用最新的版本号+1
	GenerateImagesForNarration(ctx context.Context, narrationID string) ([]string, error)

	// GetImageVersions 获取章节的所有图片版本号
	GetImageVersions(ctx context.Context, chapterID string) ([]int, error)

	// ListImagesByNarration 获取解说的图片列表（可指定版本；version<=0 则取最新版本）
	ListImagesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Image, int, error)
}

// GenerateImagesForNarration 为章节解说生成所有章节图片
// version: 图片版本号，如果为空则自动生成下一个版本号（基于该章节已有的图片版本），如果指定则自动生成下一个版本号
func (s *novelService) GenerateImagesForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// 1. 获取章节解说
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("find narration: %w", err)
	}

	// 2. 从独立的表中查询场景和镜头
	scenes, err := s.sceneRepo.FindByNarrationID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("find scenes: %w", err)
	}

	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes found for narration")
	}

	// 2. 自动生成下一个版本号（基于章节ID，独立递增）
	imageVersion, err := s.getNextImageVersion(ctx, narration.ChapterID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get next image version: %w", err)
	}

	// 2. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, narration.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	// 3. 同步角色信息到小说级别
	if err := s.SyncCharactersFromNarration(ctx, chapter.NovelID, narrationID); err != nil {
		return nil, fmt.Errorf("sync characters: %w", err)
	}

	// 4. 获取小说的所有角色
	characters, err := s.GetCharactersByNovelID(ctx, chapter.NovelID)
	if err != nil {
		return nil, fmt.Errorf("get characters: %w", err)
	}

	// 构建角色映射
	characterMap := make(map[string]*novel.Character)
	for _, char := range characters {
		characterMap[char.Name] = char
	}

	// 5. 获取图片生成提供者（初始化时已创建）
	imageProvider := s.imageProvider

	// 6. 初始化 Prompt 构建器
	promptBuilder := noveltools.NewImagePromptBuilder()

	// 7. 遍历所有场景和镜头，生成图片
	var imageIDs []string
	sequence := 1

	for _, scene := range scenes {
		// 查询该场景下的所有镜头
		shots, err := s.shotRepo.FindBySceneID(ctx, scene.ID)
		if err != nil {
			log.Error().
				Err(err).
				Str("scene_id", scene.ID).
				Msg("查询镜头失败，跳过该场景")
			continue
		}

		for _, shot := range shots {
			// 查找角色信息
			character, ok := characterMap[shot.Character]
			if !ok {
				log.Warn().
					Str("character", shot.Character).
					Str("scene", scene.SceneNumber).
					Str("shot", shot.ShotNumber).
					Msg("角色信息未找到，跳过")
				continue
			}

			// 生成单张图片
			imageID, err := s.generateSingleImage(
				ctx,
				narration,
				chapter,
				scene,
				shot,
				character,
				imageProvider,
				promptBuilder,
				sequence,
				imageVersion,
			)
			if err != nil {
				log.Error().
					Err(err).
					Str("scene", scene.SceneNumber).
					Str("shot", shot.ShotNumber).
					Msg("生成图片失败")
				continue
			}

			imageIDs = append(imageIDs, imageID)
			sequence++
		}
	}

	return imageIDs, nil
}

// generateSingleChapterImage 生成单张章节图片（私有方法）
func (s *novelService) generateSingleImage(
	ctx context.Context,
	narration *novel.Narration,
	chapter *novel.Chapter,
	scene *novel.Scene,
	shot *novel.Shot,
	character *novel.Character,
	imageProvider noveltools.ImageProvider,
	promptBuilder *noveltools.ImagePromptBuilder,
	sequence int,
	version int,
) (string, error) {
	// 1. 构建完整 prompt
	completePrompt := promptBuilder.BuildCompletePrompt(character, shot.ScenePrompt)

	// 2. 构建输出文件名
	outputFilename := fmt.Sprintf("chapter_%03d_image_%02d.jpeg", chapter.Sequence, sequence)

	// 3. 使用图片生成提供者生成图片
	imageData, err := imageProvider.GenerateImage(ctx, completePrompt, outputFilename)
	if err != nil {
		return "", fmt.Errorf("generate image: %w", err)
	}

	// 8. 上传图片到 resource 模块
	uploadReq := &service.UploadFileRequest{
		UserID:      narration.UserID,
		FileName:    outputFilename,
		ContentType: "image/jpeg",
		Ext:         "jpeg",
		Data:        bytes.NewReader(imageData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}

	// 9. 保存 ChapterImage 记录
	imageID := id.New()
	chapterImage := &novel.Image{
		ID:              imageID,
		ChapterID:       chapter.ID,
		NarrationID:     narration.ID,
		WorkflowID:      chapter.WorkflowID,
		SceneNumber:     scene.SceneNumber,
		ShotNumber:      shot.ShotNumber,
		ImageResourceID: uploadResult.ResourceID,
		CharacterName:   shot.Character,
		Prompt:          completePrompt,
		Version:         version, // 使用指定的版本号
		Status:          novel.TaskStatusCompleted,
		Sequence:        sequence,
	}

	if err := s.imageRepo.Create(ctx, chapterImage); err != nil {
		return "", fmt.Errorf("create chapter image: %w", err)
	}

	log.Info().
		Str("image_id", imageID).
		Str("chapter_id", chapter.ID).
		Str("scene", scene.SceneNumber).
		Str("shot", shot.ShotNumber).
		Msg("章节图片生成成功")

	return imageID, nil
}

// getNextImageVersion 获取章节的下一个图片版本号（自动递增）
// chapterID: 章节ID
// baseVersion: 基础版本号（如 1），如果为0则自动生成下一个版本号
func (s *novelService) getNextImageVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	versions, err := s.imageRepo.FindVersionsByChapterID(ctx, chapterID)
	if err != nil {
		// 如果没有找到任何版本，返回 1 或基础版本号
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	if len(versions) == 0 {
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	// 如果指定了基础版本号，检查该版本是否已存在
	if baseVersion > 0 {
		for _, v := range versions {
			if v == baseVersion {
				// 该版本已存在，返回下一个版本号
				maxVersion := 0
				for _, v := range versions {
					if v > maxVersion {
						maxVersion = v
					}
				}
				return maxVersion + 1, nil
			}
		}
		// 该版本不存在，直接返回
		return baseVersion, nil
	}

	// 如果没有指定基础版本号，查找所有版本号中的最大值
	maxVersion := 0
	for _, v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}

	return maxVersion + 1, nil
}
