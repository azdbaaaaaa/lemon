package novel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
)

// NarrationService 章节解说服务接口
// 定义章节解说相关的能力
type NarrationService interface {
	// GenerateNarrationForChapter 为单一章节生成解说文本
	GenerateNarrationForChapter(ctx context.Context, chapterID string) (string, error)

	// GenerateNarrationForChapterWithMeta 为单一章节生成解说文本，并返回本次生成的 Narration 元数据
	GenerateNarrationForChapterWithMeta(ctx context.Context, chapterID string) (*novel.Narration, string, error)

	// GenerateNarrationsForAllChapters 并发地为所有章节生成解说文本
	GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error

	// GetNarration 根据章节ID获取章节解说（返回最新版本）
	GetNarration(ctx context.Context, chapterID string) (*novel.Narration, error)

	// GetNarrationByVersion 根据章节ID和版本号获取章节解说
	GetNarrationByVersion(ctx context.Context, chapterID string, version int) (*novel.Narration, error)

	// SetNarrationVersion 设置章节解说的版本号
	SetNarrationVersion(ctx context.Context, narrationID string, version int) error

	// GetNarrationVersions 获取章节的所有版本号
	GetNarrationVersions(ctx context.Context, chapterID string) ([]int, error)

	// ListNarrationsByChapterID 列出章节的所有解说版本（包含 narration_id/version/prompt/status）
	ListNarrationsByChapterID(ctx context.Context, chapterID string) ([]*novel.Narration, error)

	// CreateNarrationVersionFromText 人工提交解说 JSON，生成新的解说版本（会写入 narrations/scenes/shots）
	CreateNarrationVersionFromText(ctx context.Context, chapterID, userID, prompt, narrationText string) (*novel.Narration, error)

	// GetScenesByNarrationID 获取解说对应的场景列表（用于人工编辑/比对）
	GetScenesByNarrationID(ctx context.Context, narrationID string) ([]*novel.Scene, error)

	// GetShotsByNarrationID 获取解说对应的镜头列表（用于人工编辑/比对）
	GetShotsByNarrationID(ctx context.Context, narrationID string) ([]*novel.Shot, error)

	// UpdateShot 更新分镜头信息
	UpdateShot(ctx context.Context, shotID string, updates map[string]interface{}) error

	// RegenerateShotScript 重新生成单个分镜头的脚本（调用 LLM）
	RegenerateShotScript(ctx context.Context, shotID string) error
}

// GenerateNarrationForChapterWithMeta 为单一章节生成章节解说，并保存到 narrations/scenes/shots 表
func (s *novelService) GenerateNarrationForChapterWithMeta(ctx context.Context, chapterID string) (*novel.Narration, string, error) {
	return s.generateNarrationForChapter(ctx, chapterID)
}

// GenerateNarrationForChapter 为单一章节生成章节解说，并保存到 chapter_narrations 表
// 返回的是 JSON 格式的字符串，实际存储的是结构化数据
func (s *novelService) GenerateNarrationForChapter(ctx context.Context, chapterID string) (string, error) {
	n, txt, err := s.generateNarrationForChapter(ctx, chapterID)
	if err != nil {
		return "", err
	}
	_ = n
	return txt, nil
}

func (s *novelService) generateNarrationForChapter(ctx context.Context, chapterID string) (*novel.Narration, string, error) {
	startTime := time.Now()
	log.Info().
		Str("chapter_id", chapterID).
		Msg("开始生成章节剧本")

	ch, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		log.Error().Err(err).Str("chapter_id", chapterID).Msg("获取章节信息失败")
		return nil, "", err
	}

	log.Debug().
		Str("chapter_id", chapterID).
		Str("novel_id", ch.NovelID).
		Int("sequence", ch.Sequence).
		Int("word_count", ch.WordCount).
		Msg("章节信息获取成功")

	totalChapters, err := s.getTotalChapters(ctx, ch.NovelID)
	if err != nil {
		log.Error().Err(err).Str("novel_id", ch.NovelID).Msg("获取章节总数失败")
		return nil, "", err
	}

	log.Debug().
		Str("chapter_id", chapterID).
		Int("total_chapters", totalChapters).
		Msg("准备生成剧本 JSON")

	prompt, filteredNarration, jsonContent, err := s.buildNarrationJSON(ctx, ch, totalChapters)
	if err != nil {
		log.Error().Err(err).
			Str("chapter_id", chapterID).
			Int("sequence", ch.Sequence).
			Msg("生成剧本 JSON 失败")
		return nil, "", err
	}

	log.Info().
		Str("chapter_id", chapterID).
		Int("scenes_count", len(jsonContent.Scenes)).
		Int("total_shots", s.countTotalShots(jsonContent)).
		Msg("剧本 JSON 生成成功")

	nextVersion, err := s.getNextNarrationVersion(ctx, ch.ID)
	if err != nil {
		log.Error().Err(err).Str("chapter_id", chapterID).Msg("获取下一个版本号失败")
		return nil, "", fmt.Errorf("failed to get next version: %w", err)
	}

	log.Debug().
		Str("chapter_id", chapterID).
		Int("version", nextVersion).
		Msg("准备保存剧本数据")

	narrationEntity, err := s.persistNarrationBatch(ctx, ch, nextVersion, prompt, jsonContent)
	if err != nil {
		log.Error().Err(err).
			Str("chapter_id", chapterID).
			Int("version", nextVersion).
			Msg("保存剧本数据失败")
		return nil, "", err
	}

	duration := time.Since(startTime)
	log.Info().
		Str("chapter_id", chapterID).
		Str("narration_id", narrationEntity.ID).
		Int("version", nextVersion).
		Int("scenes_count", len(jsonContent.Scenes)).
		Dur("duration", duration).
		Msg("章节剧本生成完成")

	return narrationEntity, filteredNarration, nil
}

// countTotalShots 统计总镜头数
func (s *novelService) countTotalShots(jsonContent *noveltools.NarrationJSONContent) int {
	count := 0
	for _, scene := range jsonContent.Scenes {
		if scene != nil {
			count += len(scene.Shots)
		}
	}
	return count
}

func (s *novelService) buildNarrationJSON(
	ctx context.Context,
	ch *novel.Chapter,
	totalChapters int,
) (prompt string, filteredNarration string, jsonContent *noveltools.NarrationJSONContent, err error) {
	log.Debug().
		Str("chapter_id", ch.ID).
		Int("sequence", ch.Sequence).
		Int("word_count", ch.WordCount).
		Msg("开始调用 LLM 生成剧本")

	llmStartTime := time.Now()
	generator := noveltools.NewNarrationGenerator(s.llmProvider)
	prompt, narrationText, err := generator.GenerateWithPrompt(ctx, ch.ChapterText, ch.Sequence, totalChapters, ch.WordCount)
	if err != nil {
		log.Error().Err(err).
			Str("chapter_id", ch.ID).
			Dur("duration", time.Since(llmStartTime)).
			Msg("LLM 生成剧本失败")
		return "", "", nil, err
	}

	llmDuration := time.Since(llmStartTime)
	log.Info().
		Str("chapter_id", ch.ID).
		Int("narration_length", len(narrationText)).
		Dur("llm_duration", llmDuration).
		Msg("LLM 生成剧本完成")

	narrationText = strings.TrimSpace(narrationText)
	if narrationText == "" {
		log.Error().
			Str("chapter_id", ch.ID).
			Msg("LLM 返回的剧本内容为空")
		return "", "", nil, fmt.Errorf("generated narrationText is empty")
	}

	log.Debug().
		Str("chapter_id", ch.ID).
		Msg("开始审核和过滤剧本内容")

	filteredNarration, err = s.auditAndFilterNarration(ctx, narrationText, ch.Sequence)
	if err != nil {
		log.Warn().Err(err).
			Str("chapter_id", ch.ID).
			Msg("审核和过滤剧本内容失败，使用原始内容")
		filteredNarration = narrationText
	} else {
		log.Debug().
			Str("chapter_id", ch.ID).
			Msg("剧本内容审核和过滤完成")
	}

	log.Debug().
		Str("chapter_id", ch.ID).
		Msg("开始解析剧本 JSON")

	parseStartTime := time.Now()
	jsonContent, err = noveltools.ParseNarrationJSON(filteredNarration)
	if err != nil {
		log.Error().Err(err).
			Str("chapter_id", ch.ID).
			Dur("duration", time.Since(parseStartTime)).
			Msg("解析剧本 JSON 失败")
		return "", "", nil, fmt.Errorf("narration parsing failed: %w", err)
	}

	if len(jsonContent.Scenes) == 0 {
		log.Error().
			Str("chapter_id", ch.ID).
			Msg("剧本 JSON 验证失败：缺少 scenes 字段或 scenes 为空")
		return "", "", nil, fmt.Errorf("narration validation failed: 缺少 scenes 字段或 scenes 为空")
	}

	parseDuration := time.Since(parseStartTime)
	log.Info().
		Str("chapter_id", ch.ID).
		Int("scenes_count", len(jsonContent.Scenes)).
		Int("total_shots", s.countTotalShots(jsonContent)).
		Dur("parse_duration", parseDuration).
		Msg("剧本 JSON 解析成功")

	return prompt, filteredNarration, jsonContent, nil
}

func (s *novelService) persistNarrationBatch(
	ctx context.Context,
	ch *novel.Chapter,
	version int,
	prompt string,
	jsonContent *noveltools.NarrationJSONContent,
) (*novel.Narration, error) {
	persistStartTime := time.Now()
	narrationID := id.New()

	log.Debug().
		Str("chapter_id", ch.ID).
		Str("narration_id", narrationID).
		Int("version", version).
		Msg("开始保存剧本数据")

	narrationEntity := &novel.Narration{
		ID:        narrationID,
		ChapterID: ch.ID,
		NovelID:   ch.NovelID,
		UserID:    ch.UserID,
		Prompt:    prompt,
		Version:   version,
		Status:    novel.TaskStatusPending, // 初始状态为 pending，成功后再更新为 completed
	}
	if err := s.narrationRepo.Create(ctx, narrationEntity); err != nil {
		log.Error().Err(err).
			Str("chapter_id", ch.ID).
			Str("narration_id", narrationID).
			Msg("创建解说记录失败")
		return nil, fmt.Errorf("failed to create narration record: %w", err)
	}

	log.Debug().
		Str("narration_id", narrationID).
		Msg("开始转换场景和镜头数据")

	// 转换场景、镜头、角色和道具
	convertStartTime := time.Now()
	scenes, shots, characters, props, err := noveltools.ConvertToScenesAndShots(narrationID, ch.ID, ch.NovelID, ch.UserID, version, jsonContent)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to convert scenes and shots: %v", err)
		log.Error().Err(err).
			Str("narration_id", narrationID).
			Dur("duration", time.Since(convertStartTime)).
			Msg("转换场景和镜头数据失败")
		// 更新状态为失败
		_ = s.narrationRepo.UpdateStatus(ctx, narrationID, novel.TaskStatusFailed, errorMsg)
		return nil, fmt.Errorf("failed to convert scenes and shots: %w", err)
	}

	convertDuration := time.Since(convertStartTime)
	log.Info().
		Str("narration_id", narrationID).
		Int("scenes_count", len(scenes)).
		Int("shots_count", len(shots)).
		Int("characters_count", len(characters)).
		Int("props_count", len(props)).
		Dur("convert_duration", convertDuration).
		Msg("场景、镜头、角色和道具数据转换完成")

	// 保存场景
	if len(scenes) > 0 {
		log.Debug().
			Str("narration_id", narrationID).
			Int("scenes_count", len(scenes)).
			Msg("开始保存场景数据")

		saveScenesStartTime := time.Now()
		if err := s.sceneRepo.CreateMany(ctx, scenes); err != nil {
			errorMsg := fmt.Sprintf("failed to save scenes: %v", err)
			log.Error().Err(err).
				Str("narration_id", narrationID).
				Int("scenes_count", len(scenes)).
				Dur("duration", time.Since(saveScenesStartTime)).
				Msg("保存场景数据失败")
			// 更新状态为失败
			_ = s.narrationRepo.UpdateStatus(ctx, narrationID, novel.TaskStatusFailed, errorMsg)
			return nil, fmt.Errorf("failed to save scenes: %w", err)
		}

		saveScenesDuration := time.Since(saveScenesStartTime)
		log.Info().
			Str("narration_id", narrationID).
			Int("scenes_count", len(scenes)).
			Dur("save_duration", saveScenesDuration).
			Msg("场景数据保存完成")
	}

	// 保存镜头
	if len(shots) > 0 {
		log.Debug().
			Str("narration_id", narrationID).
			Int("shots_count", len(shots)).
			Msg("开始保存镜头数据")

		saveShotsStartTime := time.Now()
		if err := s.shotRepo.CreateMany(ctx, shots); err != nil {
			errorMsg := fmt.Sprintf("failed to save shots: %v", err)
			log.Error().Err(err).
				Str("narration_id", narrationID).
				Int("shots_count", len(shots)).
				Dur("duration", time.Since(saveShotsStartTime)).
				Msg("保存镜头数据失败")
			// 更新状态为失败
			_ = s.narrationRepo.UpdateStatus(ctx, narrationID, novel.TaskStatusFailed, errorMsg)
			return nil, fmt.Errorf("failed to save shots: %w", err)
		}

		saveShotsDuration := time.Since(saveShotsStartTime)
		log.Info().
			Str("narration_id", narrationID).
			Int("shots_count", len(shots)).
			Dur("save_duration", saveShotsDuration).
			Msg("镜头数据保存完成")
	}

	// 保存角色（去重：如果角色已存在，则更新；否则创建）
	if len(characters) > 0 {
		log.Debug().
			Str("narration_id", narrationID).
			Int("characters_count", len(characters)).
			Msg("开始保存角色数据")

		saveCharsStartTime := time.Now()
		for _, char := range characters {
			// 检查角色是否已存在
			existing, err := s.characterRepo.FindByNameAndNovelID(ctx, char.Name, ch.NovelID)
			if err == nil && existing != nil {
				// 角色已存在，更新信息（保留已有的图片等）
				updates := bson.M{
					"updated_at": time.Now(),
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
				char.CreatedAt = time.Now()
				char.UpdatedAt = time.Now()
				if err := s.characterRepo.Create(ctx, char); err != nil {
					log.Warn().Err(err).
						Str("character_name", char.Name).
						Msg("创建角色失败，继续处理")
				}
			}
		}

		saveCharsDuration := time.Since(saveCharsStartTime)
		log.Info().
			Str("narration_id", narrationID).
			Int("characters_count", len(characters)).
			Dur("save_duration", saveCharsDuration).
			Msg("角色数据保存完成")
	}

	// 保存道具（去重：如果道具已存在，则更新；否则创建）
	if len(props) > 0 {
		log.Debug().
			Str("narration_id", narrationID).
			Int("props_count", len(props)).
			Msg("开始保存道具数据")

		savePropsStartTime := time.Now()
		for _, prop := range props {
			// 检查道具是否已存在
			existing, err := s.propRepo.FindByName(ctx, ch.NovelID, prop.Name)
			if err == nil && existing != nil {
				// 道具已存在，更新信息（保留已有的图片等）
				updates := map[string]interface{}{
					"updated_at": time.Now(),
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
				prop.CreatedAt = time.Now()
				prop.UpdatedAt = time.Now()
				if err := s.propRepo.Create(ctx, prop); err != nil {
					log.Warn().Err(err).
						Str("prop_name", prop.Name).
						Msg("创建道具失败，继续处理")
				}
			}
		}

		savePropsDuration := time.Since(savePropsStartTime)
		log.Info().
			Str("narration_id", narrationID).
			Int("props_count", len(props)).
			Dur("save_duration", savePropsDuration).
			Msg("道具数据保存完成")
	}

	// 所有操作成功，更新状态为 completed
	if err := s.narrationRepo.UpdateStatus(ctx, narrationID, novel.TaskStatusCompleted, ""); err != nil {
		log.Error().Err(err).
			Str("narration_id", narrationID).
			Msg("更新解说状态失败")
		return nil, fmt.Errorf("failed to update narration status: %w", err)
	}
	narrationEntity.Status = novel.TaskStatusCompleted

	persistDuration := time.Since(persistStartTime)
	log.Info().
		Str("narration_id", narrationID).
		Str("chapter_id", ch.ID).
		Int("version", version).
		Int("scenes_count", len(scenes)).
		Int("shots_count", len(shots)).
		Dur("persist_duration", persistDuration).
		Msg("剧本数据保存完成")

	return narrationEntity, nil
}

// GenerateNarrationsForAllChapters 第三步：并发地根据每一章节内容生成章节对应的章节解说
func (s *novelService) GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error {
	log.Info().
		Str("novel_id", novelID).
		Msg("开始为所有章节生成剧本")

	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		log.Error().Err(err).Str("novel_id", novelID).Msg("获取章节列表失败")
		return fmt.Errorf("failed to find chapters: %w", err)
	}
	if len(chapters) == 0 {
		log.Warn().Str("novel_id", novelID).Msg("未找到章节")
		return fmt.Errorf("no chapters found for novelID=%s", novelID)
	}

	totalChapters := len(chapters)
	log.Info().
		Str("novel_id", novelID).
		Int("total_chapters", totalChapters).
		Msg("准备并发生成所有章节的剧本")

	var wg sync.WaitGroup
	errCh := make(chan error, totalChapters)

	for _, ch := range chapters {
		wg.Add(1)
		go func(chapter *novel.Chapter) {
			defer wg.Done()

			log.Debug().
				Str("chapter_id", chapter.ID).
				Int("sequence", chapter.Sequence).
				Int("word_count", chapter.WordCount).
				Msg("开始生成章节剧本")

			generator := noveltools.NewNarrationGenerator(s.llmProvider)
			// 传递章节字数，用于根据章节长度调整 prompt 要求
			llmStartTime := time.Now()
			prompt, narrationText, err := generator.GenerateWithPrompt(ctx, chapter.ChapterText, chapter.Sequence, totalChapters, chapter.WordCount)
			if err != nil {
				log.Error().Err(err).
					Str("chapter_id", chapter.ID).
					Int("sequence", chapter.Sequence).
					Dur("duration", time.Since(llmStartTime)).
					Msg("LLM 生成章节剧本失败")
				errCh <- fmt.Errorf("failed to generate narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			llmDuration := time.Since(llmStartTime)
			log.Info().
				Str("chapter_id", chapter.ID).
				Int("sequence", chapter.Sequence).
				Int("narration_length", len(narrationText)).
				Dur("llm_duration", llmDuration).
				Msg("LLM 生成章节剧本完成")

			narrationText = strings.TrimSpace(narrationText)
			if narrationText == "" {
				log.Error().
					Str("chapter_id", chapter.ID).
					Int("sequence", chapter.Sequence).
					Msg("LLM 返回的剧本内容为空")
				errCh <- fmt.Errorf("generated narrationText is empty for chapter %d", chapter.Sequence)
				return
			}

			// 步骤1: 内容审查和过滤（参考 Python 的 audit_and_filter_narration）
			// 极度宽松模式：仅提示，不阻断
			filteredNarration, err := s.auditAndFilterNarration(ctx, narrationText, chapter.Sequence)
			if err != nil {
				// 即使审查出错，也继续使用原始内容（极度宽松模式）
				filteredNarration = narrationText
			}

			// 步骤2: 解析 JSON 格式并验证
			parseStartTime := time.Now()
			jsonContent, err := noveltools.ParseNarrationJSON(filteredNarration)
			if err != nil {
				log.Error().Err(err).
					Str("chapter_id", chapter.ID).
					Int("sequence", chapter.Sequence).
					Dur("duration", time.Since(parseStartTime)).
					Msg("解析章节剧本 JSON 失败")
				errCh <- fmt.Errorf("failed to parse narration for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 基本验证：至少要有场景
			if len(jsonContent.Scenes) == 0 {
				log.Error().
					Str("chapter_id", chapter.ID).
					Int("sequence", chapter.Sequence).
					Msg("剧本 JSON 验证失败：缺少 scenes 字段或 scenes 为空")
				errCh <- fmt.Errorf("failed to validate narration for chapter %d: 缺少 scenes 字段或 scenes 为空", chapter.Sequence)
				return
			}

			parseDuration := time.Since(parseStartTime)
			log.Info().
				Str("chapter_id", chapter.ID).
				Int("sequence", chapter.Sequence).
				Int("scenes_count", len(jsonContent.Scenes)).
				Int("total_shots", s.countTotalShots(jsonContent)).
				Dur("parse_duration", parseDuration).
				Msg("章节剧本 JSON 解析成功")

			// 生成下一个版本号（自动递增）
			nextVersion, err := s.getNextNarrationVersion(ctx, chapter.ID)
			if err != nil {
				errCh <- fmt.Errorf("failed to get next version for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 创建 Narration 记录（作为本次解说生成的批次标识）
			narrationID := id.New()
			narrationEntity := &novel.Narration{
				ID:        narrationID,
				ChapterID: chapter.ID,
				NovelID:   chapter.NovelID,
				UserID:    chapter.UserID,
				Prompt:    prompt,
				Version:   nextVersion,
				Status:    novel.TaskStatusCompleted,
			}
			if err := s.narrationRepo.Create(ctx, narrationEntity); err != nil {
				errCh <- fmt.Errorf("failed to create narration record for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 步骤3: 将场景、镜头、角色和道具转换为实体并保存到独立的表中
			scenes, shots, characters, props, err := noveltools.ConvertToScenesAndShots(narrationID, chapter.ID, chapter.NovelID, chapter.UserID, nextVersion, jsonContent)
			if err != nil {
				errCh <- fmt.Errorf("failed to convert scenes and shots for chapter %d: %w", chapter.Sequence, err)
				return
			}

			// 批量保存场景
			if len(scenes) > 0 {
				if err := s.sceneRepo.CreateMany(ctx, scenes); err != nil {
					errCh <- fmt.Errorf("failed to save scenes for chapter %d: %w", chapter.Sequence, err)
					return
				}
			}

			// 批量保存镜头
			if len(shots) > 0 {
				if err := s.shotRepo.CreateMany(ctx, shots); err != nil {
					errCh <- fmt.Errorf("failed to save shots for chapter %d: %w", chapter.Sequence, err)
					return
				}
			}

			// 保存角色（去重：如果角色已存在，则更新；否则创建）
			for _, char := range characters {
				existing, err := s.characterRepo.FindByNameAndNovelID(ctx, char.Name, chapter.NovelID)
				if err == nil && existing != nil {
					// 角色已存在，更新信息
					updates := bson.M{}
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
					if len(updates) > 0 {
						_ = s.characterRepo.Update(ctx, existing.ID, updates)
					}
				} else {
					// 角色不存在，创建新角色
					char.CreatedAt = time.Now()
					char.UpdatedAt = time.Now()
					_ = s.characterRepo.Create(ctx, char)
				}
			}

			// 保存道具（去重：如果道具已存在，则更新；否则创建）
			for _, prop := range props {
				existing, err := s.propRepo.FindByName(ctx, chapter.NovelID, prop.Name)
				if err == nil && existing != nil {
					// 道具已存在，更新信息
					updates := map[string]interface{}{}
					if prop.Description != "" {
						updates["description"] = prop.Description
					}
					if prop.ImagePrompt != "" {
						updates["image_prompt"] = prop.ImagePrompt
					}
					if prop.Category != "" {
						updates["category"] = prop.Category
					}
					if len(updates) > 0 {
						_ = s.propRepo.Update(ctx, existing.ID, updates)
					}
				} else {
					// 道具不存在，创建新道具
					prop.CreatedAt = time.Now()
					prop.UpdatedAt = time.Now()
					_ = s.propRepo.Create(ctx, prop)
				}
			}
		}(ch)
	}

	wg.Wait()
	close(errCh)

	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		log.Error().
			Str("novel_id", novelID).
			Int("total_chapters", totalChapters).
			Int("failed_count", len(errors)).
			Int("success_count", totalChapters-len(errors)).
			Msg("部分章节剧本生成失败")
		return fmt.Errorf("failed to generate narrations for %d chapters: %v", len(errors), errors)
	}

	log.Info().
		Str("novel_id", novelID).
		Int("total_chapters", totalChapters).
		Msg("所有章节剧本生成完成")

	return nil
}

// GetNarration 根据章节ID获取章节解说（返回最新版本）
func (s *novelService) GetNarration(ctx context.Context, chapterID string) (*novel.Narration, error) {
	return s.narrationRepo.FindByChapterID(ctx, chapterID)
}

// GetNarrationByVersion 根据章节ID和版本号获取章节解说
func (s *novelService) GetNarrationByVersion(ctx context.Context, chapterID string, version int) (*novel.Narration, error) {
	return s.narrationRepo.FindByChapterIDAndVersion(ctx, chapterID, version)
}

// SetNarrationVersion 设置章节解说的版本号
func (s *novelService) SetNarrationVersion(ctx context.Context, narrationID string, version int) error {
	return s.narrationRepo.UpdateVersion(ctx, narrationID, version)
}

// GetNarrationVersions 获取章节的所有版本号
// 注意：现在从 Scene 表中获取版本号，因为不再使用 Narration 表
func (s *novelService) GetNarrationVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.narrationRepo.FindVersionsByChapterID(ctx, chapterID)
}

func (s *novelService) ListNarrationsByChapterID(ctx context.Context, chapterID string) ([]*novel.Narration, error) {
	return s.narrationRepo.FindAllByChapterID(ctx, chapterID)
}

func (s *novelService) CreateNarrationVersionFromText(
	ctx context.Context,
	chapterID, userID, prompt, narrationText string,
) (*novel.Narration, error) {
	ch, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	narrationText = strings.TrimSpace(narrationText)
	if narrationText == "" {
		return nil, fmt.Errorf("narrationText is empty")
	}

	jsonContent, err := noveltools.ParseNarrationJSON(narrationText)
	if err != nil {
		return nil, fmt.Errorf("narration parsing failed: %w", err)
	}
	if len(jsonContent.Scenes) == 0 {
		return nil, fmt.Errorf("narration validation failed: 缺少 scenes 字段或 scenes 为空")
	}

	nextVersion, err := s.getNextNarrationVersion(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version: %w", err)
	}

	// userID 以请求为准（运营可能代操作），但 chapter.UserID 用于后续资源归属
	if userID != "" {
		ch.UserID = userID
	}

	narrationEntity, err := s.persistNarrationBatch(ctx, ch, nextVersion, prompt, jsonContent)
	if err != nil {
		return nil, err
	}
	return narrationEntity, nil
}

func (s *novelService) GetScenesByNarrationID(ctx context.Context, narrationID string) ([]*novel.Scene, error) {
	return s.sceneRepo.FindByNarrationID(ctx, narrationID)
}

func (s *novelService) GetShotsByNarrationID(ctx context.Context, narrationID string) ([]*novel.Shot, error) {
	return s.shotRepo.FindByNarrationID(ctx, narrationID)
}

// UpdateShot 更新分镜头信息
func (s *novelService) UpdateShot(ctx context.Context, shotID string, updates map[string]interface{}) error {
	return s.shotRepo.Update(ctx, shotID, updates)
}

// RegenerateShotScript 重新生成单个分镜头的脚本（调用 LLM）
func (s *novelService) RegenerateShotScript(ctx context.Context, shotID string) error {
	// 1. 获取分镜头信息
	shot, err := s.shotRepo.FindByID(ctx, shotID)
	if err != nil {
		return fmt.Errorf("find shot: %w", err)
	}

	// 2. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, shot.ChapterID)
	if err != nil {
		return fmt.Errorf("find chapter: %w", err)
	}

	// 3. 获取章节总数
	totalChapters, err := s.getTotalChapters(ctx, chapter.NovelID)
	if err != nil {
		return fmt.Errorf("get total chapters: %w", err)
	}

	// 4. 构建简单的 prompt：基于分镜头的当前信息，要求 LLM 优化
	prompt := fmt.Sprintf(`请优化以下分镜头的脚本信息：

分镜头编号：%s
当前解说：%s
当前图片提示词：%s
当前视频提示词：%s
当前运镜方式：%s
当前时长：%.1f秒

请返回优化后的 JSON 格式，只包含以下字段：
{
  "narration": "优化后的解说内容",
  "image_prompt": "优化后的图片提示词",
  "video_prompt": "优化后的视频提示词",
  "camera_movement": "优化后的运镜方式",
  "duration": 优化后的时长（秒，数字）
}

要求：
1. 只返回 JSON，不要其他文字
2. 确保 JSON 格式正确，可以直接解析
3. 优化后的内容应该更详细、更符合视频制作需求`,
		shot.ShotNumber,
		shot.Narration,
		shot.ImagePrompt,
		shot.VideoPrompt,
		shot.CameraMovement,
		shot.Duration,
	)

	// 5. 调用 LLM 生成优化后的脚本
	generator := noveltools.NewNarrationGenerator(s.llmProvider)
	_, optimizedText, err := generator.GenerateWithPrompt(ctx, prompt, chapter.Sequence, totalChapters, chapter.WordCount)
	if err != nil {
		return fmt.Errorf("generate optimized script: %w", err)
	}

	// 6. 解析 JSON（简单的解析，只提取需要的字段）
	var result struct {
		Narration      string  `json:"narration"`
		ImagePrompt    string  `json:"image_prompt"`
		VideoPrompt    string  `json:"video_prompt"`
		CameraMovement string  `json:"camera_movement"`
		Duration       float64 `json:"duration"`
	}

	cleanedText := noveltools.CleanJSONContent(optimizedText)
	if err := json.Unmarshal([]byte(cleanedText), &result); err != nil {
		return fmt.Errorf("parse optimized script: %w", err)
	}

	// 7. 更新分镜头信息
	updates := map[string]interface{}{}
	if result.Narration != "" {
		updates["narration"] = result.Narration
	}
	if result.ImagePrompt != "" {
		updates["image_prompt"] = result.ImagePrompt
	}
	if result.VideoPrompt != "" {
		updates["video_prompt"] = result.VideoPrompt
	}
	if result.CameraMovement != "" {
		updates["camera_movement"] = result.CameraMovement
	}
	if result.Duration > 0 {
		updates["duration"] = result.Duration
	}

	if len(updates) > 0 {
		if err := s.shotRepo.Update(ctx, shotID, updates); err != nil {
			return fmt.Errorf("update shot: %w", err)
		}
	}

	return nil
}

// GetAudioVersions 获取章节解说的所有音频版本号
func (s *novelService) GetAudioVersions(ctx context.Context, narrationID string) ([]int, error) {
	return s.audioRepo.FindVersionsByNarrationID(ctx, narrationID)
}

// GetSubtitleVersions 获取章节的所有字幕版本号
func (s *novelService) GetSubtitleVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.subtitleRepo.FindVersionsByChapterID(ctx, chapterID)
}

// GetImageVersions 获取章节的所有图片版本号
func (s *novelService) GetImageVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.imageRepo.FindVersionsByChapterID(ctx, chapterID)
}

// getTotalChapters 获取小说的总章节数
func (s *novelService) getTotalChapters(ctx context.Context, novelID string) (int, error) {
	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return 0, err
	}
	if len(chapters) == 0 {
		return 0, fmt.Errorf("no chapters found for novelID=%s", novelID)
	}
	return len(chapters), nil
}

// getNextNarrationVersion 获取章节的下一个版本号（自动递增）
// chapterID: 章节ID
// 例如：如果已有 1, 2，则返回 3
// 注意：现在从 Scene 表中获取版本号，因为不再使用 Narration 表
func (s *novelService) getNextNarrationVersion(ctx context.Context, chapterID string) (int, error) {
	versions, err := s.narrationRepo.FindVersionsByChapterID(ctx, chapterID)
	if err != nil || len(versions) == 0 {
		return 1, nil
	}
	maxVersion := 0
	for _, v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}
	if maxVersion == 0 {
		return 1, nil
	}
	return maxVersion + 1, nil
}

// auditAndFilterNarration 对生成的章节解说内容进行审查和过滤（极度宽松模式）
// 参考 Python 的 audit_and_filter_narration 方法
// 仅提示，不阻断，即使检测到敏感内容也返回原始内容
func (s *novelService) auditAndFilterNarration(ctx context.Context, narration string, chapterNum int) (string, error) {
	contentFilter := noveltools.NewContentFilter()

	// 检查内容是否包含违禁词汇（仅提示，不阻断）
	checkResult := contentFilter.CheckContent(narration)

	if !checkResult.IsSafe {
		// 记录警告日志（在实际环境中可以使用 log 包）
		// log.Warn().Int("chapter_num", chapterNum).Strs("issues", checkResult.Issues).
		// 	Msg("检测到敏感内容，但继续生成")
		_ = checkResult.Issues // 暂时忽略，避免未使用变量警告
	}

	// 无论是否检测到敏感内容，都返回原始内容（极度宽松模式）
	return narration, nil
}
