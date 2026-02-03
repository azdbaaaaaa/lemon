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

	prompt := buildChapterNarrationPrompt(chapterContent, chapterNum, totalChapters)
	narration, err := ng.llmProvider.Generate(ctx, prompt)
	return prompt, narration, err
}

// buildChapterNarrationPrompt 构造章节解说的提示词
// 要求生成 JSON 格式的结构化数据
func buildChapterNarrationPrompt(chapterContent string, chapterNum, totalChapters int) string {
	var b strings.Builder
	b.WriteString("你是一名专业的中文小说解说文案撰写助手。\n")
	b.WriteString("请基于下面给出的章节内容，生成适合短视频解说的结构化解说文案。\n\n")

	b.WriteString("【重要输出格式要求】\n")
	b.WriteString("1. 必须只返回纯 JSON 格式的内容，不要添加任何其他文字\n")
	b.WriteString("2. 不要使用 markdown 代码块标记（不要使用 ```json 或 ```）\n")
	b.WriteString("3. 不要添加任何解释、说明或注释\n")
	b.WriteString("4. 直接以 { 开头，以 } 结尾\n")
	b.WriteString("5. 确保 JSON 格式完全正确，可以直接被 JSON 解析器解析\n\n")

	b.WriteString("【内容要求】\n")
	b.WriteString("1. 必须生成至少7个分镜，每个分镜包含解说内容和图片描述\n")
	b.WriteString("2. 解说内容总字数必须达到1100-1300字（中文字符）\n")
	b.WriteString("3. 使用第三人称口播风格，语言自然、口语化\n")
	b.WriteString("4. 不要剧透后续章节，只围绕当前章节的内容\n\n")

	b.WriteString("【解说内容（narration）要求】\n")
	b.WriteString("1. 每个分镜的解说内容必须完整自然，能够独立成段，包含足够的信息量\n")
	b.WriteString("2. 解说内容应该只包含小说情节、对话、人物心理活动、事件描述等故事内容\n")
	b.WriteString("3. 每个特写的解说内容应该详细描述该特写对应的情节片段，包括：\n")
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
	b.WriteString("1. 每个特写必须包含一个 video_prompt 字段，用于生成该镜头的动态视频\n")
	b.WriteString("2. video_prompt 应描述该镜头的动态效果，例如：\n")
	b.WriteString("   - \"镜头缓慢推进，人物缓缓回头\"\n")
	b.WriteString("   - \"树叶随风飘动，光影斑驳\"\n")
	b.WriteString("   - \"画面有明显的动态效果，人物有自然的动作和表情变化\"\n")
	b.WriteString("   - \"镜头缓慢拉远，背景有轻微的运动感\"\n")
	b.WriteString("3. video_prompt 应该描述镜头运动、人物动作、画面变化等动态效果\n")
	b.WriteString("4. 如果没有明确的动态效果需求，可以使用默认描述：\"画面有明显的动态效果，动作大一些\"\n\n")

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
      "role_number": "角色编号"
    }
  ],
  "scenes": [
    {
      "scene_number": "1",
      "narration": "分镜级别的解说内容（可选）",
      "shots": [
        {
          "closeup_number": "1",
          "character": "特写人物姓名",
          "narration": "特写解说内容（30-32字，只包含故事内容，如：他缓缓转过身，目光中带着一丝疑惑。不要包含技术性描述）",
          "scene_prompt": "场景描述（室内/外、季节、天气等）+ 角色描述 + 行为/事件 + 构图词（镜头类型、光影、画面质量等）",
          "video_prompt": "镜头缓慢推进，人物缓缓回头，画面有明显的动态效果"
        }
      ]
    }
  ]
}`)
	b.WriteString("\n\n【再次强调】\n")
	b.WriteString("1. 只返回 JSON 内容，不要任何 markdown 代码块标记\n")
	b.WriteString("2. 不要添加任何解释文字，直接输出 JSON\n")
	b.WriteString("3. 确保解说内容总字数在1100-1300字之间，且至少有7个分镜\n")
	b.WriteString("4. 输出必须以 { 开头，以 } 结尾\n")
	b.WriteString("5. 解说内容（narration）必须只包含故事内容，禁止包含任何技术性描述（如\"室内场景\"、\"光影\"、\"近景拍摄\"等）\n")
	b.WriteString("6. 所有技术性描述必须放在 scene_prompt 和 video_prompt 字段中，不要放在 narration 中\n")

	return b.String()
}
