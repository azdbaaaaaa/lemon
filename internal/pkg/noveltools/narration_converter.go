package noveltools

import (
	"fmt"

	"lemon/internal/model/novel"
)

// ConvertToScenesAndShots 将解析后的 JSON 内容转换为 Scene、Shot、Character、Prop 实体
// 这是一个纯函数，不依赖任何 service 层状态，适合放在 pkg 包中
// narrationID 用作一次解说生成的批次标识；同一章节可有多个版本，每个版本对应一个 narrationID
func ConvertToScenesAndShots(
	narrationID string,
	chapterID string,
	novelID string,
	userID string,
	version int,
	jsonContent *NarrationJSONContent,
) ([]*novel.Scene, []*novel.Shot, []*novel.Character, []*novel.Prop, error) {
	var scenes []*novel.Scene
	var shots []*novel.Shot
	var characters []*novel.Character
	var props []*novel.Prop

	// 转换角色
	characterMap := make(map[string]*novel.Character) // 用于去重
	for _, jsonChar := range jsonContent.Characters {
		if jsonChar == nil || jsonChar.Name == "" {
			continue
		}
		// 如果角色已存在，跳过（避免重复）
		if _, exists := characterMap[jsonChar.Name]; exists {
			continue
		}
		character := &novel.Character{
			ID:          fmt.Sprintf("%s-char-%s-v%d", narrationID, jsonChar.Name, version),
			NovelID:     novelID,
			Name:        jsonChar.Name,
			Gender:      jsonChar.Gender,
			AgeGroup:    jsonChar.AgeGroup,
			RoleNumber:  jsonChar.RoleNumber,
			Description: jsonChar.Description,
			ImagePrompt: jsonChar.ImagePrompt,
			Status:      novel.TaskStatusCompleted,
		}
		characters = append(characters, character)
		characterMap[jsonChar.Name] = character
	}

	// 转换道具
	propMap := make(map[string]*novel.Prop) // 用于去重
	for _, jsonProp := range jsonContent.Props {
		if jsonProp == nil || jsonProp.Name == "" {
			continue
		}
		// 如果道具已存在，跳过（避免重复）
		if _, exists := propMap[jsonProp.Name]; exists {
			continue
		}
		prop := &novel.Prop{
			ID:          fmt.Sprintf("%s-prop-%s-v%d", narrationID, jsonProp.Name, version),
			NovelID:     novelID,
			Name:        jsonProp.Name,
			Description: jsonProp.Description,
			ImagePrompt: jsonProp.ImagePrompt,
			Category:    jsonProp.Category,
			Status:      novel.TaskStatusCompleted,
		}
		props = append(props, prop)
		propMap[jsonProp.Name] = prop
	}

	globalShotIndex := 1 // 全局镜头索引（在所有镜头中的顺序，从1开始）

	for sceneSeq, jsonScene := range jsonContent.Scenes {
		if jsonScene == nil {
			continue
		}

		// 创建 Scene 实体
		sceneID := fmt.Sprintf("%s-scene-%s-v%d", narrationID, jsonScene.SceneNumber, version)
		scene := &novel.Scene{
			ID:          sceneID,
			NarrationID: narrationID,
			ChapterID:   chapterID,
			NovelID:     novelID,
			UserID:      userID,
			SceneNumber: jsonScene.SceneNumber,
			Description: jsonScene.Description,
			ImagePrompt: jsonScene.ImagePrompt,
			Narration:   jsonScene.Narration,
			Sequence:    sceneSeq + 1, // 从1开始
			Version:     version,
			Status:      novel.TaskStatusCompleted,
		}
		scenes = append(scenes, scene)

		// 创建该场景下的所有 Shot 实体
		for shotSeq, jsonShot := range jsonScene.Shots {
			if jsonShot == nil {
				continue
			}

			shotID := fmt.Sprintf("%s-shot-%s-%s-v%d", narrationID, jsonScene.SceneNumber, jsonShot.CloseupNumber, version)
			shot := &novel.Shot{
				ID:          shotID,
				SceneID:     sceneID,
				SceneNumber: jsonScene.SceneNumber,
				NarrationID: narrationID,
				ChapterID:   chapterID,
				NovelID:     novelID,
				UserID:      userID,
				ShotNumber:     jsonShot.CloseupNumber,
				Character:      jsonShot.Character,
				Image:          jsonShot.Image,
				Narration:      jsonShot.Narration,
				SoundEffect:    jsonShot.SoundEffect,
				Duration:       jsonShot.Duration,
				ImagePrompt:    jsonShot.ImagePrompt,
				VideoPrompt:    jsonShot.VideoPrompt,
				CameraMovement: jsonShot.CameraMovement,
				Sequence:       shotSeq + 1,     // 在场景中的顺序，从1开始
				Index:          globalShotIndex, // 全局索引
				Version:        version,
				Status:         novel.TaskStatusCompleted,
			}
			shots = append(shots, shot)
			globalShotIndex++
		}
	}

	return scenes, shots, characters, props, nil
}
