package noveltools

import (
	"fmt"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
)

// ConvertToScenesAndShots 将解析后的 JSON 内容转换为 Scene、Shot、Character、Prop 实体
// 这是一个纯函数，不依赖任何 service 层状态，适合放在 pkg 包中
func ConvertToScenesAndShots(
	chapterID string,
	novelID string,
	userID string,
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
			ID:          id.New(),
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
			ID:          id.New(),
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

	for sceneSeq, jsonScene := range jsonContent.Scenes {
		if jsonScene == nil {
			continue
		}

		// 创建 Scene 实体
		sceneID := fmt.Sprintf("scene-%d", sceneSeq+1)
		scene := &novel.Scene{
			ID:          sceneID,
			ChapterID:   chapterID,
			NovelID:     novelID,
			UserID:      userID,
			Description: jsonScene.Description,
			Sequence:    sceneSeq + 1, // 从1开始
			Status:      novel.TaskStatusCompleted,
		}
		scenes = append(scenes, scene)

		// 创建该场景下的所有 Shot 实体
		for shotSeq, jsonShot := range jsonScene.Shots {
			if jsonShot == nil {
				continue
			}

			shotID := fmt.Sprintf("shot-%d-%d", sceneSeq+1, shotSeq+1)
			shot := &novel.Shot{
				ID:               shotID,
				SceneID:          sceneID,
				ChapterID:        chapterID,
				NovelID:          novelID,
				UserID:           userID,
				Narration:        jsonShot.Narration,
				Duration:         jsonShot.Duration,
				FirstImagePrompt: jsonShot.FirstImagePrompt,
				LastImagePrompt:  jsonShot.LastImagePrompt,
				VideoPrompt:      jsonShot.VideoPrompt,
				Sequence:         shotSeq + 1, // 在场景中的顺序，从1开始
				Status:           novel.TaskStatusCompleted,
			}
			shots = append(shots, shot)
		}
	}

	return scenes, shots, characters, props, nil
}
