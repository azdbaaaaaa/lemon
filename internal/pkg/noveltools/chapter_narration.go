package noveltools

import (
	"context"
	"fmt"
	"strings"
)

// LLMProvider 定义了调用大模型的接口
// 具体的「如何调用大模型」由调用方通过实现此接口注入，方便单测和替换实现
type LLMProvider interface {
	// Generate 根据提示词生成文本
	//
	// Args:
	//   - ctx: 上下文
	//   - prompt: 提示词
	//
	// Returns:
	//   - text: 生成的文本
	//   - err: 错误信息
	Generate(ctx context.Context, prompt string) (string, error)
}

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
	if ng.llmProvider == nil {
		return "", fmt.Errorf("llmProvider is required")
	}
	chapterContent = strings.TrimSpace(chapterContent)
	if chapterContent == "" {
		return "", fmt.Errorf("chapterContent is empty")
	}
	if chapterNum <= 0 || totalChapters <= 0 {
		return "", fmt.Errorf("invalid chapter number or totalChapters")
	}

	prompt := buildChapterNarrationPrompt(chapterContent, chapterNum, totalChapters)
	return ng.llmProvider.Generate(ctx, prompt)
}

// buildChapterNarrationPrompt 构造章节解说的提示词
//
// 这里先用简单模板实现，后续可以改为基于文件的模板（类似 Python 版本的 Jinja2）。
func buildChapterNarrationPrompt(chapterContent string, chapterNum, totalChapters int) string {
	var b strings.Builder
	b.WriteString("你是一名专业的中文小说解说文案撰写助手。\n")
	b.WriteString("请基于下面给出的章节内容，生成适合短视频解说的中文文案。\n")
	b.WriteString("要求：\n")
	b.WriteString("1. 使用第三人称口播风格，语言自然、口语化。\n")
	b.WriteString("2. 保持情节完整，控制在 200~400 字左右。\n")
	b.WriteString("3. 不要剧透后续章节，只围绕当前章节的内容。\n")
	b.WriteString("4. 可以适当加入氛围渲染，但不要加入无关剧情。\n\n")

	fmt.Fprintf(&b, "当前进度：第 %d 章 / 共 %d 章。\n\n", chapterNum, totalChapters)
	b.WriteString("下面是本章节的原始内容：\n")
	b.WriteString("---- BEGIN CHAPTER ----\n")
	b.WriteString(chapterContent)
	b.WriteString("\n---- END CHAPTER ----\n\n")
	b.WriteString("现在请根据以上内容，输出一段解说文案（只输出文案本身，不要解释）：\n")

	return b.String()
}
