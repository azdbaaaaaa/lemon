package noveltools

import (
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// mockLLMProvider 用于测试的 mock LLM 提供者
type mockLLMProvider struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *mockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "", errors.New("mock generate function not set")
}

func TestNarrationGenerator_Generate(t *testing.T) {
	Convey("NarrationGenerator.Generate 能正确生成解说文案", t, func() {
		ctx := context.Background()

		Convey("正常生成解说", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					So(prompt, ShouldContainSubstring, "你是一名专业的中文小说解说文案撰写助手")
					So(prompt, ShouldContainSubstring, "第一章内容")
					So(prompt, ShouldContainSubstring, "第 1 章 / 共 10 章")
					return "这是生成的解说文案", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "第一章内容", 1, 10)

			So(err, ShouldBeNil)
			So(result, ShouldEqual, "这是生成的解说文案")
		})

		Convey("llmProvider 为 nil 时应返回错误", func() {
			generator := &NarrationGenerator{llmProvider: nil}
			result, err := generator.Generate(ctx, "章节内容", 1, 10)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "llmProvider is required")
			So(result, ShouldBeEmpty)
		})

		Convey("章节内容为空时应返回错误", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "", 1, 10)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "chapterContent is empty")
			So(result, ShouldBeEmpty)
		})

		Convey("章节内容只有空白字符时应返回错误", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "   \n\n  ", 1, 10)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "chapterContent is empty")
			So(result, ShouldBeEmpty)
		})

		Convey("章节编号为 0 时应返回错误", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "章节内容", 0, 10)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid chapter number")
			So(result, ShouldBeEmpty)
		})

		Convey("章节编号为负数时应返回错误", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "章节内容", -1, 10)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid chapter number")
			So(result, ShouldBeEmpty)
		})

		Convey("总章节数为 0 时应返回错误", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "章节内容", 1, 0)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid chapter number")
			So(result, ShouldBeEmpty)
		})

		Convey("总章节数为负数时应返回错误", func() {
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "章节内容", 1, -1)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid chapter number")
			So(result, ShouldBeEmpty)
		})

		Convey("llmProvider 返回错误时应传递错误", func() {
			expectedErr := errors.New("LLM API error")
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return "", expectedErr
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			result, err := generator.Generate(ctx, "章节内容", 1, 10)

			So(err, ShouldEqual, expectedErr)
			So(result, ShouldBeEmpty)
		})

		Convey("prompt 应包含正确的章节信息", func() {
			var capturedPrompt string
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					capturedPrompt = prompt
					return "解说文案", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			_, err := generator.Generate(ctx, "这是第一章的详细内容", 5, 20)

			So(err, ShouldBeNil)
			So(capturedPrompt, ShouldContainSubstring, "第 5 章 / 共 20 章")
			So(capturedPrompt, ShouldContainSubstring, "这是第一章的详细内容")
			So(capturedPrompt, ShouldContainSubstring, "---- BEGIN CHAPTER ----")
			So(capturedPrompt, ShouldContainSubstring, "---- END CHAPTER ----")
		})

		Convey("章节内容会被自动去除首尾空白", func() {
			var capturedPrompt string
			mockProvider := &mockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					capturedPrompt = prompt
					return "解说文案", nil
				},
			}

			generator := NewNarrationGenerator(mockProvider)
			_, err := generator.Generate(ctx, "  \n章节内容\n  ", 1, 10)

			So(err, ShouldBeNil)
			// 验证 prompt 中的内容已经去除首尾空白
			lines := strings.Split(capturedPrompt, "\n")
			var foundContent bool
			for _, line := range lines {
				if strings.Contains(line, "章节内容") {
					foundContent = true
					So(line, ShouldNotContainSubstring, "  \n")
					So(line, ShouldNotContainSubstring, "\n  ")
				}
			}
			So(foundContent, ShouldBeTrue)
		})
	})
}

func TestBuildChapterNarrationPrompt(t *testing.T) {
	Convey("buildChapterNarrationPrompt 能正确构建提示词", t, func() {
		Convey("提示词应包含所有必要元素", func() {
			prompt := buildChapterNarrationPrompt("章节内容", 3, 10)

			So(prompt, ShouldContainSubstring, "你是一名专业的中文小说解说文案撰写助手")
			So(prompt, ShouldContainSubstring, "第 3 章 / 共 10 章")
			So(prompt, ShouldContainSubstring, "章节内容")
			So(prompt, ShouldContainSubstring, "---- BEGIN CHAPTER ----")
			So(prompt, ShouldContainSubstring, "---- END CHAPTER ----")
			So(prompt, ShouldContainSubstring, "第三人称口播风格")
			So(prompt, ShouldContainSubstring, "200~400 字")
		})

		Convey("提示词格式应正确", func() {
			prompt := buildChapterNarrationPrompt("测试内容", 1, 5)

			// 验证章节内容被正确包裹
			beginIdx := strings.Index(prompt, "---- BEGIN CHAPTER ----")
			endIdx := strings.Index(prompt, "---- END CHAPTER ----")
			So(beginIdx, ShouldBeGreaterThan, -1)
			So(endIdx, ShouldBeGreaterThan, beginIdx)
			So(prompt[beginIdx+len("---- BEGIN CHAPTER ----"):endIdx], ShouldContainSubstring, "测试内容")
		})
	})
}
