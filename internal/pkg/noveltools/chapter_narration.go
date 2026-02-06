package noveltools

import (
	"context"
	"fmt"
	"strings"
)

// NarrationGenerator 解说文案生成器，用于为章节生成解说文案
//
// 设计原则：
//   - 不负责落库 / 不依赖 HTTP / 不操作资源，只负责组装 prompt 并调用上层注入的 LLM 客户端
//   - 具体的「如何调用大模型」由调用方通过 llmProvider 注入，方便单测和替换实现
type NarrationGenerator struct {
	llmProvider LLMProvider // 调用大模型的提供者（由上层注入，便于在不同环境下切换实现）
}

// NewNarrationGenerator 创建解说文案生成器实例
//
// Args:
//   - llmProvider: 调用大模型的提供者（由上层注入，便于在不同环境下切换实现）
//
// Returns:
//   - *NarrationGenerator: 生成器实例
func NewNarrationGenerator(llmProvider LLMProvider) *NarrationGenerator {
	return &NarrationGenerator{
		llmProvider: llmProvider,
	}
}

// Generate 生成单章节解说
//
// Args:
//   - ctx: 上下文
//   - chapterContent: 章节原始内容
//   - chapterNum: 当前章节编号（从 1 开始）
//   - totalChapters: 总章节数
//
// Returns:
//   - narration: 大模型生成的解说文案
//   - err: 错误信息
func (ng *NarrationGenerator) Generate(
	ctx context.Context,
	chapterContent string,
	chapterNum int,
	totalChapters int,
) (string, error) {
	_, narration, err := ng.GenerateWithPrompt(ctx, chapterContent, chapterNum, totalChapters)
	return narration, err
}

// GenerateWithPrompt 生成单章节解说，并返回使用的提示词
//
// Args:
//   - ctx: 上下文
//   - chapterContent: 章节原始内容
//   - chapterNum: 当前章节编号（从 1 开始）
//   - totalChapters: 总章节数
//   - chapterWordCount: 章节字数（可选，用于调整 prompt 要求）
//
// Returns:
//   - prompt: 使用的提示词
//   - narration: 大模型生成的解说文案
//   - err: 错误信息
func (ng *NarrationGenerator) GenerateWithPrompt(
	ctx context.Context,
	chapterContent string,
	chapterNum int,
	totalChapters int,
	chapterWordCount ...int,
) (string, string, error) {
	if ng.llmProvider == nil {
		return "", "", fmt.Errorf("llmProvider is required")
	}
	chapterContent = strings.TrimSpace(chapterContent)
	if chapterContent == "" {
		return "", "", fmt.Errorf("chapterContent is empty")
	}
	if chapterNum <= 0 || totalChapters <= 0 {
		return "", "", fmt.Errorf("invalid chapter number or totalChapters")
	}

	var wordCount int
	if len(chapterWordCount) > 0 {
		wordCount = chapterWordCount[0]
	}

	prompt := buildChapterNarrationPrompt(chapterContent, chapterNum, totalChapters, wordCount)
	narration, err := ng.llmProvider.Generate(ctx, prompt)
	return prompt, narration, err
}

// buildChapterNarrationPrompt 构造章节解说的提示词
// 要求生成 JSON 格式的结构化数据
// chapterWordCount: 章节字数（可选），用于根据章节长度调整 prompt 要求
func buildChapterNarrationPrompt(chapterContent string, chapterNum, totalChapters int, chapterWordCount int) string {
	var b strings.Builder
	b.WriteString("你是一名专业的中文小说解说文案撰写助手。\n")
	b.WriteString("请基于下面给出的章节内容，生成适合短视频解说的结构化解说文案。\n\n")

	b.WriteString("【⚠️ 关键输出格式要求 - 必须严格遵守】\n")
	b.WriteString("你的输出必须是一个有效的 JSON 对象，可以直接被 JSON.parse() 或 json.Unmarshal() 解析。\n\n")

	b.WriteString("【JSON 格式检查清单 - 输出前必须逐项检查】\n")
	b.WriteString("✓ 1. 输出必须以单个左花括号 { 开头，以单个右花括号 } 结尾\n")
	b.WriteString("✓ 2. 不要使用 markdown 代码块标记（绝对不要使用 ```json 或 ```）\n")
	b.WriteString("✓ 3. 不要添加任何解释、说明、注释或额外文字，只输出 JSON\n")
	b.WriteString("✓ 4. 所有键名必须使用双引号包裹（例如：\"scene_number\"，不要使用单引号）\n")
	b.WriteString("✓ 5. 所有字符串值必须使用双引号包裹（例如：\"场景描述\"，不要使用单引号）\n")
	b.WriteString("✓ 6. **绝对禁止在数组最后一个元素后添加逗号**（错误示例：[1, 2, 3,]，正确示例：[1, 2, 3]）\n")
	b.WriteString("✓ 7. **绝对禁止在对象最后一个属性后添加逗号**（错误示例：{\"key\": \"value\",}，正确示例：{\"key\": \"value\"}）\n")
	b.WriteString("✓ 8. 不要在 JSON 中添加任何注释（JSON 标准不支持 // 或 /* */ 注释）\n")
	b.WriteString("✓ 9. 确保所有字符串中的特殊字符都已正确转义（\\n, \\t, \\\", \\\\ 等）\n")
	b.WriteString("✓ 10. 确保 JSON 结构完整，所有括号、方括号都正确配对\n")
	b.WriteString("✓ 11. 输出前请使用 JSON 验证工具检查格式，确保可以解析\n\n")

	b.WriteString("【输出格式】\n")
	b.WriteString("你的输出必须是一个完整的、有效的 JSON 对象，格式如下：\n")
	b.WriteString("{\n")
	b.WriteString("  \"chapter_info\": {...},\n")
	b.WriteString("  \"characters\": [...],\n")
	b.WriteString("  \"props\": [...],\n")
	b.WriteString("  \"scenes\": [...]\n")
	b.WriteString("}\n")
	b.WriteString("注意：最后一行 scenes 数组的最后一个元素后面不要有逗号！\n\n")

	b.WriteString("【内容要求】\n")
	b.WriteString("1. 必须生成7个场景（scene），每个场景包含1-3个分镜头（shot）\n")
	b.WriteString("2. 每个分镜头必须包含：解说内容（narration）、图片描述（scene_prompt）、视频描述（video_prompt）\n")
	b.WriteString("3. 必须提取并列出本章节中出现的所有角色（characters），包括角色的基本信息（姓名、性别、年龄段、角色编号）和详细描述（外貌、性格、背景等），以及角色图片提示词\n")
	b.WriteString("4. 必须提取并列出本章节中出现的所有重要道具（props），包括道具的名称、描述、类别（如：武器、法器、丹药、服饰等）和图片提示词\n")

	// 根据章节长度调整字数要求
	if chapterWordCount > 0 {
		// 根据章节字数动态调整解说字数要求（约为章节字数的 10-15%）
		minNarrationWords := chapterWordCount / 10
		maxNarrationWords := chapterWordCount * 15 / 100
		if minNarrationWords < 800 {
			minNarrationWords = 800
		}
		if maxNarrationWords < 1000 {
			maxNarrationWords = 1000
		}
		if minNarrationWords > 1500 {
			minNarrationWords = 1500
		}
		if maxNarrationWords > 2000 {
			maxNarrationWords = 2000
		}
		fmt.Fprintf(&b, "3. 解说内容总字数必须达到%d-%d字（中文字符，根据章节长度%d字调整）\n", minNarrationWords, maxNarrationWords, chapterWordCount)
	} else {
		b.WriteString("3. 解说内容总字数必须达到1100-1300字（中文字符）\n")
	}

	b.WriteString("4. 使用第三人称口播风格，语言自然、口语化\n")
	b.WriteString("5. 不要剧透后续章节，只围绕当前章节的内容\n\n")

	b.WriteString("【解说内容（narration）要求】\n")
	b.WriteString("1. 每个分镜头的解说内容必须完整自然，能够独立成段，包含足够的信息量\n")
	b.WriteString("2. 解说内容应该只包含小说情节、对话、人物心理活动、事件描述等故事内容\n")
	b.WriteString("3. 每个分镜头的解说内容应该详细描述该分镜头对应的情节片段，包括：\n")
	b.WriteString("   - 人物的动作、表情、心理活动\n")
	b.WriteString("   - 对话内容（如果有）\n")
	b.WriteString("   - 情节的发展和转折\n")
	b.WriteString("   - 场景氛围和情绪渲染\n")
	b.WriteString("4. 禁止在解说内容中出现技术性描述，包括但不限于：\n")
	b.WriteString("   - 禁止出现\"室内场景\"、\"室外场景\"、\"光影\"、\"近景\"、\"远景\"、\"中景\"等镜头和画面技术描述\n")
	b.WriteString("   - 禁止出现\"拍摄\"、\"镜头\"、\"画面\"、\"构图\"等影视技术词汇\n")
	b.WriteString("   - 禁止出现\"季节\"、\"天气\"等环境描述（这些应该放在 scene_prompt 中）\n")
	b.WriteString("5. 解说内容应该专注于故事本身，描述发生了什么、人物说了什么、想了什么\n")
	b.WriteString("6. 所有技术性描述（场景、镜头、光影等）应该只放在 scene_prompt 和 video_prompt 字段中\n\n")

	b.WriteString("【图片描述（scene_prompt）要求】\n")
	b.WriteString("1. 图片描述必须包含场景信息：室内/外场景的具体关键词、季节、天气等\n")
	b.WriteString("2. 图片描述必须包含画面构图：镜头类型（特写/中景/远景）、光影、画面质量等\n")
	b.WriteString("3. 图片描述应该由场景描述+角色描述+行为/事件+构图词组成\n")
	b.WriteString("4. 图片描述不能包含文字相关的描述\n")
	b.WriteString("5. 每个图片描述只能描述一个人物，禁止使用多人描述词汇\n")
	b.WriteString("6. 古代背景设定：如果小说背景设定在古代，所有图片的风格必须统一设定为宋朝风格\n\n")

	b.WriteString("【视频描述（video_prompt）要求】\n")
	b.WriteString("1. 每个分镜头必须包含一个 video_prompt 字段，用于生成该分镜头的动态视频\n")
	b.WriteString("2. video_prompt 必须包含以下信息：\n")
	b.WriteString("   - 景别：特写/中景/远景/全景等镜头类型\n")
	b.WriteString("   - 镜头运动：推进/拉远/横移/跟随/固定等运动方式\n")
	b.WriteString("   - 时长：视频时长（秒），通常根据解说内容长度确定，一般5-15秒\n")
	b.WriteString("   - 动态效果：人物动作、画面变化、光影变化等\n")
	b.WriteString("3. video_prompt 格式示例：\n")
	b.WriteString("   - \"特写镜头，缓慢推进，时长8秒，人物缓缓回头，画面有明显的动态效果\"\n")
	b.WriteString("   - \"中景镜头，固定机位，时长10秒，树叶随风飘动，光影斑驳\"\n")
	b.WriteString("   - \"远景镜头，缓慢拉远，时长12秒，背景有轻微的运动感\"\n")
	b.WriteString("   - \"特写镜头，横移跟随，时长6秒，人物有自然的动作和表情变化\"\n")
	b.WriteString("4. 如果没有明确的动态效果需求，可以使用默认描述：\"特写镜头，固定机位，时长10秒，画面有明显的动态效果，动作大一些\"\n\n")

	fmt.Fprintf(&b, "当前进度：第 %d 章 / 共 %d 章。\n\n", chapterNum, totalChapters)
	b.WriteString("下面是本章节的原始内容：\n")
	b.WriteString("---- BEGIN CHAPTER ----\n")
	b.WriteString(chapterContent)
	b.WriteString("\n---- END CHAPTER ----\n\n")

	b.WriteString("【输出格式示例】\n")
	b.WriteString("请严格按照以下 JSON 格式输出，直接输出 JSON 内容，不要任何其他文字：\n")
	b.WriteString(`{
  "chapter_info": {
    "chapter_number": `)
	fmt.Fprintf(&b, "%d", chapterNum)
	b.WriteString(`,
    "format": "章节风格（如：双时代格式、单一时代格式）",
    "paint_style": "绘画风格（如：写实风格）"
  },
  "characters": [
    {
      "name": "角色姓名",
      "gender": "男/女",
      "age_group": "青年/中年/老年/青少年/儿童",
      "role_number": "角色编号",
      "description": "角色详细描述（外貌、性格、背景等）",
      "image_prompt": "角色图片提示词（用于生成角色图片）"
    }
  ],
  "props": [
    {
      "name": "道具名称",
      "description": "道具详细描述",
      "image_prompt": "道具图片提示词（用于生成道具图片）",
      "category": "道具类别（如：武器、法器、丹药、服饰等）"
    }
  ],
  "scenes": [
    {
      "scene_number": "1",
      "narration": "场景级别的解说内容（可选）",
      "shots": [
        {
          "closeup_number": "1",
          "character": "分镜头人物姓名",
          "narration": "分镜头解说内容（只包含故事内容，如：他缓缓转过身，目光中带着一丝疑惑。不要包含技术性描述）",
          "scene_prompt": "场景描述（室内/外、季节、天气等）+ 角色描述 + 行为/事件 + 构图词（镜头类型、光影、画面质量等）",
          "video_prompt": "特写镜头，缓慢推进，时长8秒，人物缓缓回头，画面有明显的动态效果"
        },
        {
          "closeup_number": "2",
          "character": "分镜头人物姓名",
          "narration": "分镜头解说内容",
          "scene_prompt": "图片描述（用于生成图片）",
          "video_prompt": "景别+镜头运动+时长+动态效果（例如：特写镜头，横移跟随，时长6秒，画面有明显的动态效果）"
        }
      ]
    },
    {
      "scene_number": "2",
      "shots": [
        {
          "closeup_number": "1",
          "character": "分镜头人物姓名",
          "narration": "分镜头解说内容",
          "scene_prompt": "图片描述",
          "video_prompt": "景别+镜头运动+时长+动态效果（例如：中景镜头，固定机位，时长10秒，画面有明显的动态效果）"
        }
      ]
    }
  ]
}`)
	b.WriteString("\n\n【⚠️ 最终检查 - 输出前必须确认】\n")
	b.WriteString("在输出 JSON 之前，请按照以下步骤检查：\n")
	b.WriteString("1. 确认输出以 { 开头，以 } 结尾，中间没有任何其他文字\n")
	b.WriteString("2. 确认没有使用 ```json 或 ``` 等 markdown 标记\n")
	b.WriteString("3. 确认所有数组和对象的最后一个元素后都没有逗号\n")
	b.WriteString("4. 确认所有键名和字符串值都使用双引号，没有单引号\n")
	b.WriteString("5. 确认没有添加任何注释（// 或 /* */）\n")
	b.WriteString("6. 确认 JSON 结构完整，括号配对正确\n")
	b.WriteString("7. 确认可以直接被 JSON 解析器解析（建议在输出前用 JSON 验证工具测试）\n\n")

	b.WriteString("【内容要求】\n")
	b.WriteString("1. 必须生成7个场景（scene），每个场景包含1-3个分镜头（shot）\n")
	b.WriteString("2. 每个分镜头必须包含：narration（解说内容）、scene_prompt（图片描述）、video_prompt（视频描述）\n")

	// 根据章节长度调整字数要求提示
	if chapterWordCount > 0 {
		minNarrationWords := chapterWordCount / 10
		maxNarrationWords := chapterWordCount * 15 / 100
		if minNarrationWords < 800 {
			minNarrationWords = 800
		}
		if maxNarrationWords < 1000 {
			maxNarrationWords = 1000
		}
		if minNarrationWords > 1500 {
			minNarrationWords = 1500
		}
		if maxNarrationWords > 2000 {
			maxNarrationWords = 2000
		}
		fmt.Fprintf(&b, "6. 确保解说内容总字数在%d-%d字之间（根据章节长度%d字调整）\n", minNarrationWords, maxNarrationWords, chapterWordCount)
	} else {
		b.WriteString("6. 确保解说内容总字数在1100-1300字之间\n")
	}

	b.WriteString("8. 解说内容（narration）必须只包含故事内容，禁止包含任何技术性描述（如\"室内场景\"、\"光影\"、\"近景拍摄\"等）\n")
	b.WriteString("9. 所有技术性描述必须放在 scene_prompt 和 video_prompt 字段中，不要放在 narration 中\n\n")

	b.WriteString("【JSON 格式示例 - 注意最后没有逗号】\n")
	b.WriteString("正确的格式示例（注意 scenes 数组最后一个元素后没有逗号）：\n")
	b.WriteString(`{
  "scenes": [
    {
      "scene_number": "1",
      "shots": [
        {
          "closeup_number": "1",
          "narration": "解说内容",
          "scene_prompt": "图片描述",
          "video_prompt": "视频描述"
        }
      ]
    },
    {
      "scene_number": "2",
      "shots": [
        {
          "closeup_number": "1",
          "narration": "解说内容",
          "scene_prompt": "图片描述",
          "video_prompt": "视频描述"
        }
      ]
    }
  ]
}`)
	b.WriteString("\n\n注意：上面示例中 scenes 数组的最后一个元素（scene_number: \"2\"）后面没有逗号！\n")
	b.WriteString("这是正确的格式。错误的格式是：\"2\" 后面有逗号，或者 shots 数组最后一个元素后有逗号。\n\n")

	b.WriteString("【最后提醒】\n")
	b.WriteString("请严格按照上述要求输出 JSON，确保格式完全正确，可以直接被 JSON 解析器解析。\n")
	b.WriteString("输出时不要添加任何前缀或后缀文字，直接输出 JSON 对象。\n")

	return b.String()
}
