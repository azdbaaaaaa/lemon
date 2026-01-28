package ark_test

import (
	"context"
	"fmt"
	"os"

	"lemon/internal/config"
	"lemon/internal/pkg/ark"
)

// ExampleNewClient 演示如何创建 Ark 客户端并进行简单调用
func ExampleNewClient() {
	// 创建配置
	// 注意：在实际使用中，应该从环境变量或配置文件读取 API Key
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // 仅用于示例
	}

	cfg := &config.AIConfig{
		Provider: "ark",
		APIKey:   apiKey,
		Model:    "doubao-seed-1-6-flash-250615",
		BaseURL:  "https://ark.cn-beijing.volces.com/api/v3",
	}

	// 创建客户端
	client, err := ark.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	// 使用客户端
	ctx := context.Background()
	result, err := client.CreateChatCompletionSimple(ctx, "你好，请介绍一下自己")
	if err != nil {
		fmt.Printf("调用失败: %v\n", err)
		return
	}

	fmt.Println(result)
	// Output: (实际输出取决于 API 响应)
}

// ExampleClient_CreateChatCompletion 演示如何使用完整的 ChatCompletion API
func ExampleClient_CreateChatCompletion() {
	// 从环境变量获取 API Key
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // 仅用于示例
	}

	cfg := &config.AIConfig{
		APIKey: apiKey,
		Model:  "doubao-seed-1-6-flash-250615",
	}

	client, err := ark.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	ctx := context.Background()

	maxTokens := 32 * 1024
	temperature := 0.7

	req := &ark.ChatCompletionRequest{
		Model: "doubao-seed-1-6-flash-250615",
		Messages: []ark.Message{
			{
				Role:    "user",
				Content: "请生成一个章节的解说文案，要求包含至少7个分镜",
			},
		},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		fmt.Printf("调用失败: %v\n", err)
		return
	}

	if len(resp.Choices) > 0 {
		fmt.Println("生成的文案：")
		fmt.Println(resp.Choices[0].Message.Content)

		if resp.Usage != nil {
			fmt.Printf("\nToken 使用情况：\n")
			fmt.Printf("Prompt Tokens: %d\n", resp.Usage.PromptTokens)
			fmt.Printf("Completion Tokens: %d\n", resp.Usage.CompletionTokens)
			fmt.Printf("Total Tokens: %d\n", resp.Usage.TotalTokens)
		}
	}
	// Output: (实际输出取决于 API 响应)
}

// ExampleClient_CreateChatCompletion_withSystemMessage 演示如何使用系统消息
func ExampleClient_CreateChatCompletion_withSystemMessage() {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here"
	}

	cfg := &config.AIConfig{
		APIKey: apiKey,
		Model:  "doubao-seed-1-6-flash-250615",
	}

	client, err := ark.NewClient(cfg)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	ctx := context.Background()

	req := &ark.ChatCompletionRequest{
		Model: "doubao-seed-1-6-flash-250615",
		Messages: []ark.Message{
			{
				Role:    "system",
				Content: "你是一个专业的文案生成助手，擅长创作视频解说文案。",
			},
			{
				Role:    "user",
				Content: "请为第一章生成解说文案",
			},
		},
		MaxTokens:   intPtr(2048),
		Temperature: float64Ptr(0.8),
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		fmt.Printf("调用失败: %v\n", err)
		return
	}

	if len(resp.Choices) > 0 {
		fmt.Println(resp.Choices[0].Message.Content)
	}
	// Output: (实际输出取决于 API 响应)
}

// Helper functions for examples
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
