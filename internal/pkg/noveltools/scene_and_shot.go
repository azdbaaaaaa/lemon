package noveltools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
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
	_, narration, err := ng.GenerateWithPrompt(ctx, chapterContent, chapterNum, totalChapters, 0, novel.NovelStyleMixed)
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
//   - novelStyle: 小说风格（anime/live/mixed）
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
	chapterWordCount int,
	novelStyle novel.NovelStyle,
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

	prompt := buildScenesAndShotsPrompt(chapterContent, chapterNum, totalChapters, chapterWordCount, novelStyle)
	narration, err := ng.llmProvider.Generate(ctx, prompt)
	return prompt, narration, err
}

// buildScenesAndShotsPrompt 构造场景和镜头的提示词
// 要求生成 JSON 格式的结构化数据，主要生成场景和镜头
// chapterWordCount: 章节字数，用于根据章节长度调整 prompt 要求
// novelStyle: 小说风格（anime/live/mixed），用于在提示词中添加相应的风格要求
func buildScenesAndShotsPrompt(chapterContent string, chapterNum, totalChapters int, chapterWordCount int, novelStyle novel.NovelStyle) string {
	var b strings.Builder
	b.WriteString("你是一名专业的中文小说场景和镜头生成助手。\n")
	b.WriteString("请基于下面给出的章节内容，生成适合短视频制作的场景和镜头结构。\n\n")

	b.WriteString("【⚠️ 关键输出格式要求 - 必须严格遵守】\n")
	b.WriteString("你的输出必须是一个有效的 JSON 对象，可以直接被 JSON.parse() 或 json.Unmarshal() 解析。\n\n")

	b.WriteString("【🚫 严格禁止的行为】\n")
	b.WriteString("1. **绝对禁止使用 markdown 代码块标记**（不要使用 ```json 或 ``` 包裹内容）\n")
	b.WriteString("2. **绝对禁止添加任何前缀或后缀文字**（不要添加\"以下是JSON\"、\"输出如下\"等说明文字）\n")
	b.WriteString("3. **绝对禁止添加任何注释**（JSON 标准不支持 // 或 /* */ 注释）\n")
	b.WriteString("4. **绝对禁止在数组最后一个元素后添加逗号**（错误：[1, 2, 3,]，正确：[1, 2, 3]）\n")
	b.WriteString("5. **绝对禁止在对象最后一个属性后添加逗号**（错误：{\"key\": \"value\",}，正确：{\"key\": \"value\"}）\n")
	b.WriteString("6. **绝对禁止使用单引号**（所有键名和字符串值必须使用双引号）\n\n")

	b.WriteString("【✅ 必须严格遵守的格式要求】\n")
	b.WriteString("1. 输出必须以单个左花括号 { 开头，以单个右花括号 } 结尾\n")
	b.WriteString("2. 输出必须是纯 JSON，中间不能有任何其他文字、说明或解释\n")
	b.WriteString("3. 所有键名必须使用双引号包裹（例如：\"scene_number\"）\n")
	b.WriteString("4. 所有字符串值必须使用双引号包裹（例如：\"场景描述\"）\n")
	b.WriteString("5. 所有字符串中的特殊字符必须正确转义（换行用 \\n，引号用 \\\"，反斜杠用 \\\\）\n")
	b.WriteString("6. JSON 结构必须完整，所有括号、方括号必须正确配对\n")
	b.WriteString("7. 数字类型必须是数字，不要用引号包裹（duration: 8.0 而不是 \"8.0\"）\n\n")

	b.WriteString("【📋 输出前自检清单】\n")
	b.WriteString("在输出前，请逐项检查：\n")
	b.WriteString("□ 输出是否以 { 开头，以 } 结尾？\n")
	b.WriteString("□ 是否完全没有使用 ```json 或 ``` 标记？\n")
	b.WriteString("□ 是否完全没有添加任何说明文字？\n")
	b.WriteString("□ 所有数组最后一个元素后是否都没有逗号？\n")
	b.WriteString("□ 所有对象最后一个属性后是否都没有逗号？\n")
	b.WriteString("□ 所有键名和字符串值是否都使用双引号？\n")
	b.WriteString("□ 所有特殊字符是否都已正确转义？\n")
	b.WriteString("□ JSON 结构是否完整，括号是否配对？\n")
	b.WriteString("□ 是否可以直接被 JSON 解析器解析？\n\n")

	b.WriteString("【输出格式】\n")
	b.WriteString("你的输出必须是一个完整的、有效的 JSON 对象，格式如下：\n")
	b.WriteString("{\n")
	b.WriteString("  \"characters\": [...],\n")
	b.WriteString("  \"props\": [...],\n")
	b.WriteString("  \"scenes\": [...]\n")
	b.WriteString("}\n")
	b.WriteString("注意：最后一行 scenes 数组的最后一个元素后面不要有逗号！\n\n")

	b.WriteString("【内容要求】\n")
	b.WriteString("1. 必须生成10个场景（scene），每个场景包含1-4个镜头（shot）\n")
	b.WriteString("2. 每个镜头必须包含：解说内容（narration）、首图提示词（first_image_prompt）、末图提示词（last_image_prompt）、视频提示词（video_prompt）、时长（duration）\n")
	b.WriteString("3. 每个场景必须包含：场景描述（description）、场景序号（sequence）\n")
	b.WriteString("4. 必须提取并列出本章节中出现的所有角色（characters），包括角色的基本信息（姓名、性别、年龄段、角色编号）和详细描述（外貌、性格、背景等），以及角色图片提示词（image_prompt，字数要求：不少于 100 字）\n")
	b.WriteString("5. 必须提取并列出本章节中出现的所有重要道具（props），包括道具的名称、描述、类别（如：武器、法器、丹药、服饰等）和图片提示词（image_prompt，字数要求：不少于 100 字）\n")

	// 根据章节长度调整字数要求
	var minNarrationWords, maxNarrationWords int
	if chapterWordCount > 0 {
		// 根据章节字数动态调整解说字数要求（约为章节字数的 10-15%）
		minNarrationWords = chapterWordCount / 10
		maxNarrationWords = chapterWordCount * 15 / 100
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
		minNarrationWords = 1100
		maxNarrationWords = 1300
		b.WriteString("3. 解说内容总字数必须达到1100-1300字（中文字符）\n")
	}

	// 计算每个镜头的平均字数要求（假设10个场景，平均每个场景2.5个镜头，共25个镜头）
	avgShotsPerScene := 2.5
	totalShots := 10 * avgShotsPerScene
	avgWordsPerShot := float64(minNarrationWords+maxNarrationWords) / 2.0 / totalShots
	minWordsPerShot := int(avgWordsPerShot * 0.7) // 允许 -30% 偏差
	maxWordsPerShot := int(avgWordsPerShot * 1.3) // 允许 +30% 偏差
	if minWordsPerShot < 30 {
		minWordsPerShot = 30
	}
	if maxWordsPerShot < 50 {
		maxWordsPerShot = 50
	}
	fmt.Fprintf(&b, "4. 每个镜头的解说内容字数应控制在%d-%d字之间，不要偏差太大（平均约%.0f字/镜头）\n", minWordsPerShot, maxWordsPerShot, avgWordsPerShot)

	b.WriteString("5. 使用第三人称口播风格，语言自然、口语化\n")
	b.WriteString("6. 不要剧透后续章节，只围绕当前章节的内容\n\n")

	b.WriteString("【解说内容（narration）要求】\n")
	// 使用之前计算的每个镜头字数要求
	fmt.Fprintf(&b, "1. **每个镜头的解说内容字数必须严格控制在%d-%d字之间**（平均约%.0f字/镜头），不要偏差太大\n", minWordsPerShot, maxWordsPerShot, avgWordsPerShot)
	b.WriteString("2. 每个分镜头的解说内容必须完整自然，能够独立成段，包含足够的信息量\n")
	b.WriteString("3. 解说内容应该只包含小说情节、对话、人物心理活动、事件描述等故事内容\n")
	b.WriteString("4. 每个分镜头的解说内容应该详细描述该分镜头对应的情节片段，包括：\n")
	b.WriteString("   - 人物的动作、表情、心理活动\n")
	b.WriteString("   - 对话内容（如果有）\n")
	b.WriteString("   - 情节的发展和转折\n")
	b.WriteString("   - 场景氛围和情绪渲染\n")
	b.WriteString("5. 禁止在解说内容中出现技术性描述，包括但不限于：\n")
	b.WriteString("   - 禁止出现\"室内场景\"、\"室外场景\"、\"光影\"、\"近景\"、\"远景\"、\"中景\"等镜头和画面技术描述\n")
	b.WriteString("   - 禁止出现\"拍摄\"、\"镜头\"、\"画面\"、\"构图\"等影视技术词汇\n")
	b.WriteString("   - 禁止出现\"季节\"、\"天气\"等环境描述（这些应该放在 first_image_prompt 和 last_image_prompt 中）\n")
	b.WriteString("6. 解说内容应该专注于故事本身，描述发生了什么、人物说了什么、想了什么\n")
	b.WriteString("7. 所有技术性描述（场景、镜头、光影等）应该只放在 first_image_prompt、last_image_prompt 和 video_prompt 字段中\n\n")

	b.WriteString("【首图提示词（first_image_prompt）要求】\n")
	b.WriteString("1. 首图提示词用于 AI 生成封面图，必须使用详细、可直接用于图像生成模型的提示词\n")
	b.WriteString("2. 描述必须包含以下所有要素（缺一不可）：\n")
	b.WriteString("   - 人物：角色身份、特征\n")
	b.WriteString("   - 外貌：面部特征、身材、年龄等\n")
	b.WriteString("   - 服装：服饰样式、颜色、材质等\n")
	b.WriteString("   - 表情：面部表情、情绪状态\n")
	b.WriteString("   - 动作：身体姿态、手势、动作状态\n")
	b.WriteString("   - 场景：室内/外场景的具体描述、环境细节\n")
	b.WriteString("   - 时间：时间段（如：清晨、正午、黄昏、夜晚）、季节、天气等\n")
	b.WriteString("   - 氛围：整体氛围、情绪基调\n")
	b.WriteString("   - 构图：镜头类型（特写/中景/远景）、画面布局、视角\n")
	b.WriteString("   - 光影：光线方向、强度、阴影效果\n")
	b.WriteString("   - 画面风格：电影级封面风格\n")

	// 根据小说类型添加特定的风格要求
	switch novelStyle {
	case novel.NovelStyleAnime:
		b.WriteString("3. 风格要求：电影级封面风格，动画风格，二次元风格，日系动漫风格，具有强烈的视觉冲击力和艺术美感\n")
	case novel.NovelStyleLive:
		b.WriteString("3. 风格要求：电影级封面风格，真人拍摄风格，实景拍摄，电影级真人质感，具有强烈的真实感和代入感\n")
	case novel.NovelStyleMixed:
		b.WriteString("3. 风格要求：电影级封面风格，可以结合动画风格和真人拍摄风格，具有独特的视觉表现力\n")
	default:
		b.WriteString("3. 风格要求：电影级封面风格\n")
	}

	b.WriteString("4. 字数要求：不少于 100 字（中文字符）\n")
	b.WriteString("5. 描述要求：使用中文完整描述，不要简化成短句，要详细具体\n")
	b.WriteString("6. 首图提示词应该由场景描述+角色描述+行为/事件+构图词组成\n")
	b.WriteString("7. 首图提示词不能包含文字相关的描述\n")
	b.WriteString("8. 每个首图提示词只能描述一个人物，禁止使用多人描述词汇\n")
	b.WriteString("9. 古代背景设定：如果小说背景设定在古代，所有图片的风格必须统一设定为宋朝风格\n\n")

	b.WriteString("【末图提示词（last_image_prompt）要求】\n")
	b.WriteString("1. 末图提示词用于 AI 生成封面图，必须使用详细、可直接用于图像生成模型的提示词\n")
	b.WriteString("2. 描述必须包含以下所有要素（缺一不可）：\n")
	b.WriteString("   - 人物：角色身份、特征\n")
	b.WriteString("   - 外貌：面部特征、身材、年龄等\n")
	b.WriteString("   - 服装：服饰样式、颜色、材质等\n")
	b.WriteString("   - 表情：面部表情、情绪状态（可与首图不同，体现变化）\n")
	b.WriteString("   - 动作：身体姿态、手势、动作状态（可与首图不同，体现完成状态）\n")
	b.WriteString("   - 场景：室内/外场景的具体描述、环境细节（与首图保持一致）\n")
	b.WriteString("   - 时间：时间段（如：清晨、正午、黄昏、夜晚）、季节、天气等（与首图保持一致）\n")
	b.WriteString("   - 氛围：整体氛围、情绪基调（可与首图不同，体现变化）\n")
	b.WriteString("   - 构图：镜头类型（特写/中景/远景）、画面布局、视角\n")
	b.WriteString("   - 光影：光线方向、强度、阴影效果（可与首图不同，体现变化）\n")
	b.WriteString("   - 画面风格：电影级封面风格\n")

	// 根据小说类型添加特定的风格要求
	switch novelStyle {
	case novel.NovelStyleAnime:
		b.WriteString("3. 风格要求：电影级封面风格，动画风格，二次元风格，日系动漫风格，具有强烈的视觉冲击力和艺术美感\n")
	case novel.NovelStyleLive:
		b.WriteString("3. 风格要求：电影级封面风格，真人拍摄风格，实景拍摄，电影级真人质感，具有强烈的真实感和代入感\n")
	case novel.NovelStyleMixed:
		b.WriteString("3. 风格要求：电影级封面风格，可以结合动画风格和真人拍摄风格，具有独特的视觉表现力\n")
	default:
		b.WriteString("3. 风格要求：电影级封面风格\n")
	}

	b.WriteString("4. 字数要求：不少于 100 字（中文字符）\n")
	b.WriteString("5. 描述要求：使用中文完整描述，不要简化成短句，要详细具体\n")
	b.WriteString("6. 末图提示词描述镜头结束时的画面状态\n")
	b.WriteString("7. 末图提示词应该与首图提示词在场景和角色上保持一致，但可以有不同的动作或表情\n")
	b.WriteString("8. 末图提示词应该体现镜头结束时的画面变化或动作完成状态\n\n")

	b.WriteString("【视频描述（video_prompt）要求】\n")
	b.WriteString("1. 每个分镜头必须包含一个 video_prompt 字段，用于 AI 生成视频或分镜\n")
	b.WriteString("2. 描述要求：偏\"动态画面描述\"，具有影视感，适合 10~30 秒短剧\n")
	b.WriteString("3. 描述要求：使用中文完整描述，不要简化成短句\n")
	b.WriteString("4. 字数要求：不少于 100 字（中文字符）\n")
	b.WriteString("5. video_prompt 必须包含以下信息：\n")
	b.WriteString("   - 人物动作：详细描述人物的动作、姿态变化、动作幅度等\n")
	b.WriteString("   - 镜头运动：推进/拉远/横移/跟随/固定等运动方式，以及运动速度和节奏\n")
	b.WriteString("   - 情绪变化：人物情绪的变化过程、表情变化等\n")
	b.WriteString("   - 景别：特写/中景/远景/全景等镜头类型\n")
	b.WriteString("   - 时长：视频时长（秒），通常根据解说内容长度确定，适合 10~30 秒短剧\n")
	b.WriteString("   - 动态效果：画面变化、光影变化、环境动态等\n")

	// 根据小说类型添加特定的风格要求
	switch novelStyle {
	case novel.NovelStyleAnime:
		b.WriteString("   - 画面风格：动画风格，二次元风格，日系动漫风格，具有强烈的视觉冲击力和艺术美感\n")
	case novel.NovelStyleLive:
		b.WriteString("   - 画面风格：真人拍摄风格，实景拍摄，电影级真人质感，具有强烈的真实感和代入感\n")
	case novel.NovelStyleMixed:
		b.WriteString("   - 画面风格：可以结合动画风格和真人拍摄风格，具有独特的视觉表现力\n")
	}
	b.WriteString("6. video_prompt 格式示例：\n")
	b.WriteString("   - \"特写镜头，缓慢推进，时长15秒，人物缓缓回头，眼神从疑惑转为坚定，面部表情自然变化，画面有明显的动态效果，具有强烈的影视感\"\n")
	b.WriteString("   - \"中景镜头，横移跟随，时长20秒，人物在场景中移动，动作流畅自然，情绪从紧张逐渐放松，树叶随风飘动，光影斑驳变化，适合短剧节奏\"\n")
	b.WriteString("   - \"远景镜头，缓慢拉远，时长25秒，人物在场景中完成动作，情绪有明显变化，背景有轻微的运动感，画面层次丰富，具有电影质感\"\n")
	b.WriteString("   - \"特写镜头，固定机位，时长12秒，人物有自然的动作和表情变化，情绪从平静转为激动，画面有明显的动态效果，适合短剧表现\"\n")
	b.WriteString("7. 如果没有明确的动态效果需求，可以使用默认描述：\"特写镜头，固定机位，时长15秒，人物有自然的动作和表情变化，情绪有细微变化，画面有明显的动态效果，具有影视感，适合短剧节奏\"\n\n")

	fmt.Fprintf(&b, "当前进度：第 %d 章 / 共 %d 章。\n\n", chapterNum, totalChapters)
	b.WriteString("下面是本章节的原始内容：\n")
	b.WriteString("---- BEGIN CHAPTER ----\n")
	b.WriteString(chapterContent)
	b.WriteString("\n---- END CHAPTER ----\n\n")

	b.WriteString("【输出格式示例】\n")
	b.WriteString("请严格按照以下 JSON 格式输出，直接输出 JSON 内容，不要任何其他文字：\n")
	b.WriteString(`{
  "characters": [
    {
      "name": "角色姓名",
      "gender": "男/女",
      "age_group": "青年/中年/老年/青少年/儿童",
      "role_number": "角色编号",
      "description": "角色详细描述（外貌、性格、背景等）",
      "image_prompt": "角色图片提示词（用于生成角色图片，字数要求：不少于 100 字）"
    }
  ],
  "props": [
    {
      "name": "道具名称",
      "description": "道具详细描述",
      "image_prompt": "道具图片提示词（用于生成道具图片，字数要求：不少于 100 字）",
      "category": "道具类别（如：武器、法器、丹药、服饰等）"
    }
  ],
  "scenes": [
    {
      "description": "场景详细描述",
      "shots": [
        {
          "narration": "分镜头解说内容（只包含故事内容，如：他缓缓转过身，目光中带着一丝疑惑。不要包含技术性描述）",
          "first_image_prompt": "首图提示词（场景描述+角色描述+行为/事件+构图词，字数要求：不少于 100 字）",
          "last_image_prompt": "末图提示词（镜头结束时的画面状态，字数要求：不少于 100 字）",
          "video_prompt": "特写镜头，缓慢推进，时长8秒，人物缓缓回头，画面有明显的动态效果（字数要求：不少于 100 字）",
          "duration": 8.0
        },
        {
          "narration": "分镜头解说内容",
          "first_image_prompt": "首图提示词",
          "last_image_prompt": "末图提示词",
          "video_prompt": "景别+镜头运动+时长+动态效果（例如：特写镜头，横移跟随，时长6秒，画面有明显的动态效果）",
          "duration": 6.0
        }
      ]
    },
    {
      "description": "场景详细描述",
      "shots": [
        {
          "narration": "分镜头解说内容",
          "first_image_prompt": "首图提示词",
          "last_image_prompt": "末图提示词",
          "video_prompt": "景别+镜头运动+时长+动态效果（例如：中景镜头，固定机位，时长10秒，画面有明显的动态效果）",
          "duration": 10.0
        }
      ]
    }
  ]
}`)
	b.WriteString("\n\n【⚠️ 最终输出要求 - 再次强调】\n")
	b.WriteString("请记住：你的输出必须是纯 JSON，没有任何其他内容。\n")
	b.WriteString("正确的输出示例：\n")
	b.WriteString("{\"scenes\":[...]}\n")
	b.WriteString("错误的输出示例：\n")
	b.WriteString("```json\n{\"scenes\":[...]}\n```\n")
	b.WriteString("以下是JSON输出：\n{\"scenes\":[...]}\n\n")

	b.WriteString("【内容要求】\n")
	b.WriteString("1. 必须生成10个场景（scene），每个场景包含1-4个镜头（shot）\n")
	b.WriteString("2. 每个镜头必须包含：narration（解说内容）、first_image_prompt（首图提示词）、last_image_prompt（末图提示词）、video_prompt（视频描述）、duration（时长）\n")

	// 根据章节长度调整字数要求提示（使用之前计算的变量）
	fmt.Fprintf(&b, "6. 确保解说内容总字数在%d-%d字之间（根据章节长度%d字调整）\n", minNarrationWords, maxNarrationWords, chapterWordCount)
	fmt.Fprintf(&b, "7. **每个镜头的解说内容字数必须严格控制在%d-%d字之间**（平均约%.0f字/镜头），不要偏差太大\n", minWordsPerShot, maxWordsPerShot, avgWordsPerShot)
	b.WriteString("8. 解说内容（narration）必须只包含故事内容，禁止包含任何技术性描述（如\"室内场景\"、\"光影\"、\"近景拍摄\"等）\n")
	b.WriteString("9. 所有技术性描述必须放在 first_image_prompt、last_image_prompt 和 video_prompt 字段中，不要放在 narration 中\n")
	b.WriteString("10. **重要：生成后请检查每个镜头的解说内容字数，确保都在要求的范围内，不要偏差太大**\n\n")

	b.WriteString("【JSON 格式示例 - 注意最后没有逗号】\n")
	b.WriteString("正确的格式示例（注意 scenes 数组最后一个元素后没有逗号）：\n")
	b.WriteString(`{
  "scenes": [
    {
      "description": "场景详细描述",
      "shots": [
        {
          "narration": "解说内容",
          "first_image_prompt": "首图提示词",
          "last_image_prompt": "末图提示词",
          "video_prompt": "视频描述",
          "duration": 8.0
        }
      ]
    },
    {
      "description": "场景详细描述",
      "shots": [
        {
          "narration": "解说内容",
          "first_image_prompt": "首图提示词",
          "last_image_prompt": "末图提示词",
          "video_prompt": "视频描述",
          "duration": 8.0
        }
      ]
    }
  ]
}`)
	b.WriteString("\n\n注意：上面示例中 scenes 数组的最后一个元素后面没有逗号！\n")
	b.WriteString("这是正确的格式。错误的格式是：最后一个元素后面有逗号，或者 shots 数组最后一个元素后有逗号。\n\n")

	b.WriteString("【🎯 最终输出指令】\n")
	b.WriteString("现在请直接输出 JSON 对象，不要添加任何其他内容。\n")
	b.WriteString("你的输出应该以 { 开始，以 } 结束，中间是完整的 JSON 内容。\n")
	b.WriteString("不要使用 markdown 代码块，不要添加说明文字，只输出纯 JSON。\n")

	return b.String()
}

// SceneAndShotJSONContent 用于解析 LLM 返回的 JSON
// 直接使用 model 层的结构体嵌套，但只包含 LLM 生成的字段
type SceneAndShotJSONContent struct {
	Characters []*novel.Character `json:"characters,omitempty"` // 角色列表（可选）
	Props      []*novel.Prop      `json:"props,omitempty"`      // 道具列表（可选）
	Scenes     []*SceneWithShots  `json:"scenes"`               // 场景列表（必需），嵌套包含 Shots
}

// SceneWithShots 场景结构体，嵌套包含镜头列表
// 使用 model 层的 Scene 和 Shot 结构体，但只包含 LLM 生成的字段
type SceneWithShots struct {
	// 场景字段（只包含 LLM 生成的字段）
	Description string        `json:"description"` // 场景详细描述
	Shots       []*novel.Shot `json:"shots"`       // 镜头列表，直接使用 model 层的 Shot
}

// ParseSceneAndShotJSON 解析 JSON 格式的场景和镜头数据
// 直接解析到使用 model 层结构体嵌套的格式
func ParseSceneAndShotJSON(jsonContent string) (*SceneAndShotJSONContent, error) {
	// 清理 JSON 内容（移除首尾空白，移除 markdown 代码块标记）
	jsonContent = strings.TrimSpace(jsonContent)

	// 移除 markdown 代码块标记（如果存在）
	// 优先使用简单的字符串操作，避免正则表达式
	if strings.HasPrefix(jsonContent, "```") {
		// 移除开头的 ```json 或 ```
		if strings.HasPrefix(jsonContent, "```json") {
			jsonContent = strings.TrimPrefix(jsonContent, "```json")
		} else {
			jsonContent = strings.TrimPrefix(jsonContent, "```")
		}
		jsonContent = strings.TrimSpace(jsonContent)
	}
	if strings.HasSuffix(jsonContent, "```") {
		jsonContent = strings.TrimSuffix(jsonContent, "```")
		jsonContent = strings.TrimSpace(jsonContent)
	}

	// 检查空内容
	if jsonContent == "" {
		return nil, fmt.Errorf("json content is empty")
	}

	// 尝试解析 JSON 到结构体（直接使用 model 层的嵌套结构）
	var content SceneAndShotJSONContent
	if err := json.Unmarshal([]byte(jsonContent), &content); err != nil {
		return nil, fmt.Errorf("json parse failed: %w", err)
	}

	// 验证基本结构
	if len(content.Scenes) == 0 {
		return nil, fmt.Errorf("missing scenes field or scenes is empty")
	}

	return &content, nil
}

// convertSceneAndShotJSONToEntities 将解析后的 JSON 内容转换为 Scene、Shot、Character、Prop 实体
// 由于已经使用了 model 层的嵌套结构，这里主要是补充 ID 和关联字段
func convertSceneAndShotJSONToEntities(
	chapterID string,
	novelID string,
	userID string,
	jsonContent *SceneAndShotJSONContent,
) ([]*novel.Scene, []*novel.Shot, []*novel.Character, []*novel.Prop, error) {
	var scenes []*novel.Scene
	var shots []*novel.Shot
	var characters []*novel.Character
	var props []*novel.Prop

	// 处理角色（直接使用 model 层的结构体，只需补充 ID 和关联字段）
	characterMap := make(map[string]*novel.Character) // 用于去重
	for _, char := range jsonContent.Characters {
		if char == nil || char.Name == "" {
			continue
		}
		// 如果角色已存在，跳过（避免重复）
		if _, exists := characterMap[char.Name]; exists {
			continue
		}
		// 补充 ID 和关联字段
		char.ID = id.New()
		char.NovelID = novelID
		char.Status = novel.TaskStatusCompleted
		characters = append(characters, char)
		characterMap[char.Name] = char
	}

	// 处理道具（直接使用 model 层的结构体，只需补充 ID 和关联字段）
	propMap := make(map[string]*novel.Prop) // 用于去重
	for _, prop := range jsonContent.Props {
		if prop == nil || prop.Name == "" {
			continue
		}
		// 如果道具已存在，跳过（避免重复）
		if _, exists := propMap[prop.Name]; exists {
			continue
		}
		// 补充 ID 和关联字段
		prop.ID = id.New()
		prop.NovelID = novelID
		prop.Status = novel.TaskStatusCompleted
		props = append(props, prop)
		propMap[prop.Name] = prop
	}

	// 处理场景和镜头（使用嵌套结构）
	for sceneSeq, sceneWithShots := range jsonContent.Scenes {
		if sceneWithShots == nil {
			continue
		}

		// 创建 Scene 实体
		sceneID := fmt.Sprintf("scene-%d", sceneSeq+1)
		scene := &novel.Scene{
			ID:          sceneID,
			ChapterID:   chapterID,
			NovelID:     novelID,
			UserID:      userID,
			Description: sceneWithShots.Description,
			Sequence:    sceneSeq + 1, // 从1开始
			Status:      novel.TaskStatusCompleted,
		}
		scenes = append(scenes, scene)

		// 处理该场景下的所有 Shot（直接使用 model 层的结构体，只需补充 ID 和关联字段）
		for shotSeq, shot := range sceneWithShots.Shots {
			if shot == nil {
				continue
			}

			// 补充 ID 和关联字段
			shot.ID = fmt.Sprintf("shot-%d-%d", sceneSeq+1, shotSeq+1)
			shot.SceneID = sceneID
			shot.ChapterID = chapterID
			shot.NovelID = novelID
			shot.UserID = userID
			shot.Sequence = shotSeq + 1 // 在场景中的顺序，从1开始
			shot.Status = novel.TaskStatusCompleted
			shots = append(shots, shot)
		}
	}

	return scenes, shots, characters, props, nil
}

// GenerateScenesAndShots 生成场景和镜头，并返回解析后的实体
// 这是一个便捷方法，整合了生成、解析和转换的完整流程
func (ng *NarrationGenerator) GenerateScenesAndShots(
	ctx context.Context,
	chapterContent string,
	chapterNum int,
	totalChapters int,
	chapterWordCount int,
	chapterID string,
	novelID string,
	userID string,
	novelStyle novel.NovelStyle,
) ([]*novel.Scene, []*novel.Shot, []*novel.Character, []*novel.Prop, error) {
	// 1. 生成 JSON 文本
	_, jsonText, err := ng.GenerateWithPrompt(ctx, chapterContent, chapterNum, totalChapters, chapterWordCount, novelStyle)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("generate scenes and shots failed: %w", err)
	}

	// 2. 解析 JSON
	jsonContent, err := ParseSceneAndShotJSON(jsonText)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse json failed: %w", err)
	}

	// 3. 转换为实体
	scenes, shots, characters, props, err := convertSceneAndShotJSONToEntities(chapterID, novelID, userID, jsonContent)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("convert to entities failed: %w", err)
	}

	return scenes, shots, characters, props, nil
}
