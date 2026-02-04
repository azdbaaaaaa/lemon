package noveltools

import (
	"fmt"

	"lemon/internal/model/novel"
)

// ConvertToScenesAndShots 将解析后的 JSON 内容转换为 Scene 和 Shot 实体
// 这是一个纯函数，不依赖任何 service 层状态，适合放在 pkg 包中
func ConvertToScenesAndShots(
	narrationID string,
	chapterID string,
	userID string,
	version int,
	jsonContent *NarrationJSONContent,
) ([]*novel.Scene, []*novel.Shot, error) {
	var scenes []*novel.Scene
	var shots []*novel.Shot

	globalShotIndex := 1 // 全局镜头索引（在所有镜头中的顺序，从1开始）

	for sceneSeq, jsonScene := range jsonContent.Scenes {
		if jsonScene == nil {
			continue
		}

		// 创建 Scene 实体
		sceneID := fmt.Sprintf("%s-scene-%s", narrationID, jsonScene.SceneNumber)
		scene := &novel.Scene{
			ID:          sceneID,
			NarrationID: narrationID,
			ChapterID:   chapterID,
			UserID:      userID,
			SceneNumber: jsonScene.SceneNumber,
			Narration:   jsonScene.Narration,
			Sequence:    sceneSeq + 1, // 从1开始
			Version:     version,
			Status:      "completed",
		}
		scenes = append(scenes, scene)

		// 创建该场景下的所有 Shot 实体
		for shotSeq, jsonShot := range jsonScene.Shots {
			if jsonShot == nil {
				continue
			}

			shotID := fmt.Sprintf("%s-shot-%s-%s", narrationID, jsonScene.SceneNumber, jsonShot.CloseupNumber)
			shot := &novel.Shot{
				ID:          shotID,
				SceneID:     sceneID,
				NarrationID: narrationID,
				ChapterID:   chapterID,
				UserID:      userID,
				ShotNumber:  jsonShot.CloseupNumber,
				Character:   jsonShot.Character,
				Narration:   jsonShot.Narration,
				ScenePrompt: jsonShot.ScenePrompt,
				VideoPrompt: jsonShot.VideoPrompt,
				Sequence:    shotSeq + 1,     // 在场景中的顺序，从1开始
				Index:       globalShotIndex, // 全局索引
				Version:     version,
				Status:      "completed",
			}
			shots = append(shots, shot)
			globalShotIndex++
		}
	}

	return scenes, shots, nil
}
