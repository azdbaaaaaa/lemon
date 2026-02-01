package noveltools

import (
	"context"
	"fmt"
)

// simpleLLMProvider 用于示例的简单 LLM 提供者
type simpleLLMProvider struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (s *simpleLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if s.generateFunc != nil {
		return s.generateFunc(ctx, prompt)
	}
	return "", fmt.Errorf("generate function not set")
}

// ExampleUsage 展示如何使用 noveltools 包生成单一章节的解说
//
// 这是一个示例文件，展示完整的流程：
// 1. 内容脱敏
// 2. 章节切分（如果需要）
// 3. 生成章节解说
// 4. 验证解说内容
func ExampleUsage() {
	ctx := context.Background()

	// 假设这是从资源中读取的章节内容
	chapterContent := `第一章 开始
这是第一章的内容，讲述了主角的初始状态...`

	// ===== 步骤1: 内容脱敏 =====
	filter := NewContentFilter()
	filteredContent, checkResult := filter.ProcessContent(chapterContent)
	if !checkResult.IsSafe {
		fmt.Printf("警告：检测到敏感内容: %v\n", checkResult.Issues)
	}
	fmt.Printf("脱敏后的内容长度: %d\n", len(filteredContent))

	// ===== 步骤2: 生成章节解说 =====
	// 使用 Eino LLM 提供者（默认推荐，通过 ai/component 创建 ChatModel）
	// 示例：
	//   import (
	//       "lemon/internal/ai/component"
	//       "lemon/internal/pkg/noveltools/providers"
	//       "lemon/internal/config"
	//   )
	//   cfg := &config.AIConfig{
	//       Provider: "ark",
	//       APIKey:   "your-api-key",
	//       Model:    "doubao-seed-1-6-flash-250615",
	//   }
	//   chatModel, _ := component.NewChatModel(ctx, cfg)
	//   llmProvider := providers.NewEinoProvider(chatModel)  // 使用 EinoProvider
	//   generator := NewNarrationGenerator(llmProvider)
	//
	// 或者使用 ArkProvider：
	//   import "lemon/internal/pkg/ark"
	//   import "lemon/internal/pkg/noveltools/providers"
	//   arkClient, _ := ark.NewClient(cfg)
	//   llmProvider := providers.NewArkProvider(arkClient)

	// 这里使用一个简单的实现作为示例
	simpleProvider := &simpleLLMProvider{
		generateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "这是生成的解说文案...", nil
		},
	}
	generator := NewNarrationGenerator(simpleProvider)
	narration, err := generator.Generate(ctx, filteredContent, 1, 50)
	if err != nil {
		fmt.Printf("生成解说失败: %v\n", err)
		return
	}
	fmt.Printf("生成的解说: %s\n", narration)

	// ===== 步骤3: 验证解说内容 =====
	validator := NewNarrationValidator()
	result := validator.Validate(narration, 1100, 1300, 0) // 0 表示不验证数量

	if !result.IsValid {
		fmt.Printf("验证失败: %s\n", result.Message)
		return
	}

	// 处理警告
	for _, warning := range result.Warnings {
		fmt.Printf("警告: %s\n", warning)
	}

	// 使用修正后的内容
	fixedNarration := result.Message
	fmt.Printf("验证通过，修正后的内容: %s\n", fixedNarration)
}

// GenerateSingleChapterNarration 为单一章节生成解说的完整流程
//
// 这是一个便捷函数，整合了脱敏、生成、验证的完整流程
//
// Args:
//   - ctx: 上下文
//   - llmProvider: LLM 提供者（可使用 providers.EinoProvider 或 providers.ArkProvider）
//   - chapterContent: 章节原始内容
//   - chapterNum: 章节编号（从1开始）
//   - totalChapters: 总章节数
//   - enableDesensitize: 是否启用内容脱敏
//   - enableValidation: 是否启用内容验证
//
// Returns:
//   - narration: 生成的解说文案
//   - warnings: 警告信息列表
//   - err: 错误信息
func GenerateSingleChapterNarration(
	ctx context.Context,
	llmProvider LLMProvider,
	chapterContent string,
	chapterNum int,
	totalChapters int,
	enableDesensitize bool,
	enableValidation bool,
) (narration string, warnings []string, err error) {
	content := chapterContent

	// 步骤1: 内容脱敏（可选）
	if enableDesensitize {
		filter := NewContentFilter()
		filtered, checkResult := filter.ProcessContent(content)
		if !checkResult.IsSafe {
			warnings = append(warnings, fmt.Sprintf("检测到敏感内容: %v", checkResult.Issues))
		}
		content = filtered
	}

	// 步骤2: 生成章节解说
	generator := NewNarrationGenerator(llmProvider)
	narration, err = generator.Generate(ctx, content, chapterNum, totalChapters)
	if err != nil {
		return "", warnings, fmt.Errorf("生成解说失败: %w", err)
	}

	// 步骤3: 验证解说内容（可选）
	if enableValidation {
		validator := NewNarrationValidator()
		result := validator.Validate(narration, 1100, 1300, 0)
		if !result.IsValid {
			warnings = append(warnings, fmt.Sprintf("验证失败: %s", result.Message))
		} else {
			// 使用修正后的内容
			narration = result.Message
			warnings = append(warnings, result.Warnings...)
		}
	}

	return narration, warnings, nil
}
