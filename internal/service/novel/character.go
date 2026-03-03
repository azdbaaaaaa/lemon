package novel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
)

// CharacterService 角色服务接口
// 定义角色相关的能力
type CharacterService interface {
	// GetCharactersByNovelID 获取小说的所有角色
	GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error)

	// GetCharacterByName 根据名称获取角色
	GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error)

	// GenerateCharactersFromNovel 基于整个小说内容生成角色和图片提示词
	GenerateCharactersFromNovel(ctx context.Context, novelID string) error
}

// GetCharactersByNovelID 获取小说的所有角色
func (s *novelService) GetCharactersByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error) {
	return s.characterRepo.FindByNovelID(ctx, novelID)
}

// GetCharacterByName 根据名称获取角色
func (s *novelService) GetCharacterByName(ctx context.Context, novelID, name string) (*novel.Character, error) {
	return s.characterRepo.FindByNameAndNovelID(ctx, name, novelID)
}

// GenerateCharactersFromNovel 基于整个小说内容生成角色和图片提示词
// 只负责角色提取和落库，不处理道具
func (s *novelService) GenerateCharactersFromNovel(ctx context.Context, novelID string) error {
	jsonContent, err := s.generateCharactersAndPropsJSON(ctx, novelID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, char := range jsonContent.Characters {
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
			char.ID = id.New()
			char.NovelID = novelID
			char.CreatedAt = now
			char.UpdatedAt = now
			if err := s.characterRepo.Create(ctx, char); err != nil {
				log.Warn().Err(err).
					Str("character_name", char.Name).
					Msg("创建角色失败，继续处理")
			}
		}
	}

	log.Info().
		Str("novel_id", novelID).
		Int("characters_count", len(jsonContent.Characters)).
		Msg("从小说内容生成角色成功")

	return nil
}

// generateCharactersAndPropsJSON 基于整部小说内容调用 LLM 并解析为角色+道具 JSON
func (s *novelService) generateCharactersAndPropsJSON(ctx context.Context, novelID string) (*noveltools.CharactersAndPropsJSONContent, error) {
	novelEntity, err := s.novelRepo.FindByID(ctx, novelID)
	if err != nil {
		return nil, fmt.Errorf("find novel: %w", err)
	}

	chapters, err := s.chapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		return nil, fmt.Errorf("find chapters: %w", err)
	}
	if len(chapters) == 0 {
		return nil, fmt.Errorf("novel has no chapters")
	}

	var novelContent strings.Builder
	totalWords := 0
	for _, ch := range chapters {
		novelContent.WriteString(fmt.Sprintf("\n\n=== 第 %d 章：%s ===\n\n", ch.Sequence, ch.Title))
		novelContent.WriteString(ch.ChapterText)
		totalWords += ch.WordCount
	}

	// 优先从 Prompt 模块加载角色资产提取提示词模板
	var prompt string
	if s.promptService != nil {
		tpl, err := s.promptService.GetTemplateByTypeCode(ctx, "character", "asset_extraction", "zh-CN")
		if err == nil && tpl != nil && tpl.Content != "" {
			var b strings.Builder
			b.WriteString(tpl.Content)
			fmt.Fprintf(&b, "\n小说信息：共 %d 章，总字数约 %d 字。\n\n", len(chapters), totalWords)
			b.WriteString("下面是整个小说的内容：\n")
			b.WriteString("---- BEGIN NOVEL ----\n")
			b.WriteString(novelContent.String())
			b.WriteString("\n---- END NOVEL ----\n\n")
			prompt = b.String()
		}
	}

	// 如果 PromptService 不可用或模板缺失，则退回到内置提示词
	if prompt == "" {
		prompt = buildCharactersAndPropsPrompt(novelContent.String(), len(chapters), totalWords, novelEntity.Style)
	}

	jsonText, err := s.llmProvider.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate characters and props failed: %w", err)
	}

	jsonContent, err := noveltools.ParseCharactersAndPropsJSON(jsonText)
	if err != nil {
		return nil, fmt.Errorf("parse characters and props json failed: %w", err)
	}

	return jsonContent, nil
}

// buildCharactersAndPropsPrompt 构建提取角色和道具的提示词
func buildCharactersAndPropsPrompt(novelContent string, totalChapters int, totalWords int, novelStyle novel.NovelStyle) string {
	var b strings.Builder
	b.WriteString("你是一名专业的中文小说角色和道具提取助手。\n")
	b.WriteString("你的任务是从整个小说的内容中提取所有主要角色和重要道具，并为他们生成详细的图片提示词。\n\n")

	b.WriteString("【任务说明】\n")
	b.WriteString("1. 仔细阅读整个小说的内容\n")
	b.WriteString("2. 提取所有主要角色（主角、重要配角等）\n")
	b.WriteString("3. 提取所有重要道具（武器、法器、丹药、服饰、重要物品等）\n")
	b.WriteString("4. 为每个角色和道具生成详细的图片提示词（用于 AI 生成图片）\n\n")

	b.WriteString("【角色提取要求】\n")
	b.WriteString("1. 必须提取所有主要角色，包括：\n")
	b.WriteString("   - 主角（男/女主角）\n")
	b.WriteString("   - 重要配角\n")
	b.WriteString("   - 关键反派角色\n")
	b.WriteString("   - 其他重要角色\n")
	b.WriteString("2. 每个角色必须包含以下信息：\n")
	b.WriteString("   - 姓名（name）：角色的完整姓名\n")
	b.WriteString("   - 性别（gender）：男/女\n")
	b.WriteString("   - 年龄段（age_group）：青年/中年/老年/青少年/儿童\n")
	b.WriteString("   - 角色编号（role_number）：用于标识角色的唯一编号\n")
	b.WriteString("   - 详细描述（description）：角色的外貌、性格、背景等详细描述\n")
	b.WriteString("   - 图片提示词（image_prompt）：用于生成角色图片的详细提示词，**不少于 100 字**\n")
	b.WriteString("3. 图片提示词要求：\n")
	b.WriteString("   - 必须详细描述角色的外貌特征（面部、身材、年龄等）\n")
	b.WriteString("   - 必须详细描述角色的服装（样式、颜色、材质等）\n")
	b.WriteString("   - 必须详细描述角色的气质和风格\n")
	b.WriteString("   - 必须包含背景设定（如：古代、现代、未来等）\n")
	b.WriteString("   - 字数要求：**不少于 100 字**（中文字符）\n\n")

	b.WriteString("【道具提取要求】\n")
	b.WriteString("1. 必须提取所有重要道具，包括：\n")
	b.WriteString("   - 武器（剑、刀、枪等）\n")
	b.WriteString("   - 法器（法宝、灵器等）\n")
	b.WriteString("   - 丹药（灵丹、仙丹等）\n")
	b.WriteString("   - 服饰（特殊服装、饰品等）\n")
	b.WriteString("   - 其他重要物品\n")
	b.WriteString("2. 每个道具必须包含以下信息：\n")
	b.WriteString("   - 名称（name）：道具的完整名称\n")
	b.WriteString("   - 类别（category）：武器/法器/丹药/服饰/其他\n")
	b.WriteString("   - 详细描述（description）：道具的外观、功能、特点等详细描述\n")
	b.WriteString("   - 图片提示词（image_prompt）：用于生成道具图片的详细提示词，**不少于 100 字**\n")
	b.WriteString("3. 图片提示词要求：\n")
	b.WriteString("   - 必须详细描述道具的外观（形状、大小、颜色、材质等）\n")
	b.WriteString("   - 必须详细描述道具的细节特征\n")
	b.WriteString("   - 必须包含背景设定（如：古代、现代、未来等）\n")
	b.WriteString("   - 字数要求：**不少于 100 字**（中文字符）\n\n")

	// 根据小说类型添加特定的风格要求
	switch novelStyle {
	case novel.NovelStyleAnime:
		b.WriteString("【风格要求】\n")
		b.WriteString("1. 角色图片提示词必须包含：动画风格，二次元风格，日系动漫风格\n")
		b.WriteString("2. 道具图片提示词必须包含：动画风格，二次元风格\n")
	case novel.NovelStyleLive:
		b.WriteString("【风格要求】\n")
		b.WriteString("1. 角色图片提示词必须包含：真人拍摄风格，实景拍摄，电影级真人质感\n")
		b.WriteString("2. 道具图片提示词必须包含：真实物品风格，电影级道具质感\n")
	case novel.NovelStyleMixed:
		b.WriteString("【风格要求】\n")
		b.WriteString("1. 角色图片提示词可以结合动画风格和真人拍摄风格\n")
		b.WriteString("2. 道具图片提示词可以结合动画风格和真实物品风格\n")
	}

	fmt.Fprintf(&b, "\n小说信息：共 %d 章，总字数约 %d 字。\n\n", totalChapters, totalWords)
	b.WriteString("下面是整个小说的内容：\n")
	b.WriteString("---- BEGIN NOVEL ----\n")
	b.WriteString(novelContent)
	b.WriteString("\n---- END NOVEL ----\n\n")

	b.WriteString("【输出格式】\n")
	b.WriteString("请严格按照以下 JSON 格式输出，直接输出 JSON 内容，不要任何其他文字：\n")
	b.WriteString(`{
  "characters": [
    {
      "name": "角色姓名",
      "gender": "男/女",
      "age_group": "青年/中年/老年/青少年/儿童",
      "role_number": "角色编号",
      "description": "角色详细描述（外貌、性格、背景等）",
      "image_prompt": "角色图片提示词（字数要求：不少于 100 字）"
    }
  ],
  "props": [
    {
      "name": "道具名称",
      "description": "道具详细描述",
      "image_prompt": "道具图片提示词（字数要求：不少于 100 字）",
      "category": "道具类别（如：武器、法器、丹药、服饰等）"
    }
  ]
}`)
	b.WriteString("\n\n【⚠️ 最终输出要求】\n")
	b.WriteString("请记住：你的输出必须是纯 JSON，没有任何其他内容。\n")
	b.WriteString("不要使用 markdown 代码块，不要添加说明文字，只输出纯 JSON。\n")

	return b.String()
}
