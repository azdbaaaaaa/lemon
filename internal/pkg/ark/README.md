# Ark 客户端封装

本包提供了火山引擎 Ark API（豆包大模型）的 Go 客户端封装。

## 功能特性

- ✅ 支持聊天完成（Chat Completion）API
- ✅ 支持自定义模型、温度、MaxTokens 等参数
- ✅ 提供简化版本的快速调用方法
- ✅ 线程安全的并发调用
- ✅ 使用官方 volcengine-go-sdk

## 安装依赖

本包使用官方 volcengine-go-sdk，需要安装以下依赖：

```bash
go get github.com/volcengine/volcengine-go-sdk
```

或者直接运行：
```bash
go mod tidy
```

这将自动安装所需的依赖包。

## 使用方法

### 1. 创建客户端

```go
import "lemon/internal/pkg/ark"

// 从配置创建客户端
cfg := &config.AIConfig{
    Provider: "ark",
    APIKey:   "your-api-key",
    Model:    "doubao-seed-1-6-flash-250615", // 可选，有默认值
    BaseURL:  "https://ark.cn-beijing.volces.com/api/v3", // 可选
}

client, err := ark.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
```

### 2. 简单调用（推荐）

```go
ctx := context.Background()
prompt := "请生成一个章节的解说文案..."

result, err := client.CreateChatCompletionSimple(ctx, prompt)
if err != nil {
    log.Error().Err(err).Msg("生成失败")
    return
}

fmt.Println(result)
```

### 3. 完整调用（支持更多参数）

```go
ctx := context.Background()

maxTokens := 32 * 1024
temperature := 0.7

req := &ark.ChatCompletionRequest{
    Model: "doubao-seed-1-6-flash-250615",
    Messages: []ark.Message{
        {
            Role:    "user",
            Content: "请生成一个章节的解说文案...",
        },
    },
    MaxTokens:   &maxTokens,
    Temperature: &temperature,
}

resp, err := client.CreateChatCompletion(ctx, req)
if err != nil {
    log.Error().Err(err).Msg("生成失败")
    return
}

// 获取生成的文本
if len(resp.Choices) > 0 {
    content := resp.Choices[0].Message.Content
    fmt.Println(content)
}

// 查看 Token 使用情况
if resp.Usage != nil {
    fmt.Printf("Prompt Tokens: %d\n", resp.Usage.PromptTokens)
    fmt.Printf("Completion Tokens: %d\n", resp.Usage.CompletionTokens)
    fmt.Printf("Total Tokens: %d\n", resp.Usage.TotalTokens)
}
```

## 配置说明

### AIConfig 字段

- `Provider`: 提供者名称（用于区分不同的 AI 服务）
- `APIKey`: Ark API Key（必需）
- `Model`: 模型名称（可选，默认：`doubao-seed-1-6-flash-250615`）
- `BaseURL`: API 基础 URL（可选，默认：`https://ark.cn-beijing.volces.com/api/v3`）
- `Options`: 模型参数
  - `Temperature`: 温度参数（0.0-2.0）
  - `MaxTokens`: 最大 token 数
  - `TopP`: TopP 参数

### 环境变量配置

可以通过环境变量配置：

```bash
export LEMON_AI_API_KEY="your-api-key"
export LEMON_AI_MODEL="doubao-seed-1-6-flash-250615"
export LEMON_AI_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
```

## 与 Python 代码的对应关系

Python 代码：
```python
from ark import Ark

client = Ark(api_key=ARK_CONFIG['api_key'])
response = client.chat.completions.create(
    model=self.model,
    messages=[{"role": "user", "content": prompt}],
    max_tokens=32*1024,
    temperature=0.7
)
narration = response.choices[0].message.content
```

Go 代码：
```go
client, _ := ark.NewClient(cfg)
resp, _ := client.CreateChatCompletion(ctx, &ark.ChatCompletionRequest{
    Model: "doubao-seed-1-6-flash-250615",
    Messages: []ark.Message{
        {Role: "user", Content: prompt},
    },
    MaxTokens:   volcengine.Int(32 * 1024),
    Temperature: volcengine.Float64(0.7),
})
narration := resp.Choices[0].Message.Content
```

## 错误处理

客户端会返回详细的错误信息，包括：
- API 调用失败
- HTTP 状态码错误
- JSON 解析错误
- 响应格式错误

建议在生产环境中添加重试机制和错误日志记录。

## 注意事项

1. **API Key 安全**：不要在代码中硬编码 API Key，使用环境变量或配置文件
2. **并发安全**：客户端内部使用互斥锁保证并发安全
3. **超时设置**：HTTP 客户端默认超时 60 秒，可根据需要调整
4. **Rate Limiting**：注意 API 的速率限制，避免频繁调用

## 参考文档

- [火山引擎 Go SDK](https://github.com/volcengine/volcengine-go-sdk)
- [Ark API 文档](https://www.volcengine.com/docs/82379)
