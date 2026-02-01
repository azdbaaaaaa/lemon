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

// GenerateImagesForNarration 为解说文案生成所有场景特写图片
func (s *novelService) GenerateImagesForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// 1. 获取解说文案
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("find narration: %w", err)
	}

	if narration.Content == nil {
		return nil, fmt.Errorf("narration content is nil")
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

	// 7. 遍历所有场景和特写，生成图片
	var imageIDs []string
	sequence := 1

	for _, scene := range narration.Content.Scenes {
		for _, shot := range scene.Shots {
			// 查找角色信息
			character, ok := characterMap[shot.Character]
			if !ok {
				log.Warn().
					Str("character", shot.Character).
					Str("scene", scene.SceneNumber).
					Str("shot", shot.CloseupNumber).
					Msg("角色信息未找到，跳过")
				continue
			}

			// 生成单张图片
			imageID, err := s.generateSingleSceneShotImage(
				ctx,
				narration,
				chapter,
				scene,
				shot,
				character,
				imageProvider,
				promptBuilder,
				sequence,
			)
			if err != nil {
				log.Error().
					Err(err).
					Str("scene", scene.SceneNumber).
					Str("shot", shot.CloseupNumber).
					Msg("生成图片失败")
				continue
			}

			imageIDs = append(imageIDs, imageID)
			sequence++
		}
	}

	return imageIDs, nil
}

// generateSingleSceneShotImage 生成单张场景特写图片（私有方法）
func (s *novelService) generateSingleSceneShotImage(
	ctx context.Context,
	narration *novel.Narration,
	chapter *novel.Chapter,
	scene *novel.NarrationScene,
	shot *novel.NarrationShot,
	character *novel.Character,
	imageProvider noveltools.ImageProvider,
	promptBuilder *noveltools.ImagePromptBuilder,
	sequence int,
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

	// 9. 保存 SceneShotImage 记录
	imageID := id.New()
	sceneShotImage := &novel.SceneShotImage{
		ID:              imageID,
		ChapterID:       chapter.ID,
		NarrationID:     narration.ID,
		SceneNumber:     scene.SceneNumber,
		ShotNumber:      shot.CloseupNumber,
		ImageResourceID: uploadResult.ResourceID,
		CharacterName:   shot.Character,
		Prompt:          completePrompt,
		Status:          "completed",
		Sequence:        sequence,
	}

	if err := s.sceneShotImageRepo.Create(ctx, sceneShotImage); err != nil {
		return "", fmt.Errorf("create scene shot image: %w", err)
	}

	log.Info().
		Str("image_id", imageID).
		Str("chapter_id", chapter.ID).
		Str("scene", scene.SceneNumber).
		Str("shot", shot.CloseupNumber).
		Msg("场景特写图片生成成功")

	return imageID, nil
}
