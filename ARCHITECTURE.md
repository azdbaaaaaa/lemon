# Lemon 项目架构设计文档

## 1. 项目概述

**项目名称**: Lemon
**项目类型**: 基于 AI 的 API 服务
**技术栈**: Golang + Eino (字节 AI 框架) + Cobra

### 1.1 项目目标

构建一个可扩展、高性能的 AI API 服务，提供 LLM 对话、智能代理等 AI 能力。

### 1.2 核心特性

- 基于 Cobra 的 CLI 命令管理
- 基于 Eino 的 AI 能力编排
- RESTful API 接口
- 可插拔的组件架构
- 完善的日志与监控

### 1.3 架构风格: Clean Architecture

本项目采用 **Clean Architecture (整洁架构)** 分层设计，核心原则：
- **依赖向内**: 外层依赖内层，内层不依赖外层
- **职责分离**: 每层有明确职责，易于测试和替换
- **AI 独立能力层**: AI 能力作为独立层，与业务逻辑解耦

```
┌─────────────────────────────────────────────────────────────────┐
│                        cmd/ (入口层)                             │
│                      Cobra CLI 命令                              │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    handler/ (接口适配层)                         │
│              HTTP Handler, 请求解析, 响应封装                     │
│                    依赖: service                                 │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    service/ (业务逻辑层)                         │
│              业务流程编排, 事务管理, 业务规则                      │
│                    依赖: ai, repository                         │
└─────────────────────────────────────────────────────────────────┘
                    ↓                   ↓
┌───────────────────────────┐  ┌───────────────────────────────────┐
│    ai/ (AI 能力层)         │  │     repository/ (数据访问层)       │
│  Eino Chain/Graph/Tool    │  │       MongoDB CRUD 操作           │
│  LLM 调用, RAG, Agent     │  │         依赖: model              │
│    依赖: component        │  │                                   │
└───────────────────────────┘  └───────────────────────────────────┘
                    ↓                   ↓
┌─────────────────────────────────────────────────────────────────┐
│                      model/ (领域模型层)                         │
│                实体定义, 请求/响应结构, 值对象                     │
│                        无外部依赖                                │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    pkg/ (基础设施层)                             │
│            MongoDB Client, Redis, Logger, Errors                │
│                     外部服务封装                                 │
└─────────────────────────────────────────────────────────────────┘
```

**各层职责:**

| 层级 | 目录 | 职责 | 依赖 |
|------|------|------|------|
| 入口层 | cmd/ | CLI 命令, 应用启动 | config, server |
| 接口层 | handler/ | HTTP 路由, 参数校验 | service, model |
| 业务层 | service/ | 业务编排, 规则校验 | ai, repository |
| AI 层 | ai/ | LLM 调用, RAG, Agent | eino, model |
| 数据层 | repository/ | 数据持久化 | model, pkg/mongodb |
| 模型层 | model/ | 数据结构定义 | 无 |
| 基础设施 | pkg/ | 外部服务封装 | 第三方库 |

---

## 2. 技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 语言 | Go 1.21+ | 主要开发语言 |
| CLI | Cobra + Viper | 命令行管理 + 配置管理 |
| AI 框架 | Eino | 字节跳动 LLM 应用开发框架 |
| HTTP 框架 | Gin | 高性能 Web 框架 |
| 配置管理 | Viper | 多源配置解析 (yaml/env/flag) |
| 日志 | Zerolog | 零分配 JSON 结构化日志 |
| 数据库 | MongoDB | 文档数据库，存储对话历史、用户数据 |
| 缓存 | Redis | 会话缓存、Token 缓存、限流 |
| 向量数据库 | Milvus / Pinecone | RAG 场景 (可选) |

---

## 3. 项目目录结构

```
lemon/
├── cmd/                        # CLI 命令入口
│   ├── root.go                 # 根命令
│   ├── serve.go                # API 服务启动命令
│   ├── version.go              # 版本命令
│   └── migrate.go              # 数据库迁移命令
│
├── internal/                   # 私有应用代码
│   ├── config/                 # 配置管理
│   │   ├── config.go           # 配置结构定义
│   │   └── loader.go           # 配置加载器
│   │
│   ├── server/                 # HTTP 服务器
│   │   ├── server.go           # 服务器初始化
│   │   ├── router.go           # 路由注册
│   │   └── middleware/         # 中间件
│   │       ├── auth.go         # 认证中间件
│   │       ├── logging.go      # 日志中间件
│   │       ├── recovery.go     # 异常恢复
│   │       └── cors.go         # 跨域处理
│   │
│   ├── handler/                # HTTP 处理器
│   │   ├── chat.go             # 对话接口
│   │   ├── agent.go            # 智能代理接口
│   │   └── health.go           # 健康检查
│   │
│   ├── service/                # 业务逻辑层
│   │   ├── chat_service.go     # 对话服务
│   │   └── agent_service.go    # 代理服务
│   │
│   ├── ai/                     # AI 能力层 (Eino 集成)
│   │   ├── chain/              # Chain 编排
│   │   │   └── chat_chain.go   # 对话链
│   │   ├── graph/              # Graph 编排
│   │   │   └── agent_graph.go  # 代理图
│   │   ├── component/          # 自定义组件
│   │   │   ├── model.go        # 模型封装
│   │   │   ├── retriever.go    # 检索器
│   │   │   └── tool.go         # 工具定义
│   │   └── callback/           # 回调处理
│   │       └── logging.go      # 日志回调
│   │
│   ├── model/                  # 数据模型
│   │   ├── request.go          # 请求模型
│   │   ├── response.go         # 响应模型
│   │   └── entity.go           # 业务实体
│   │
│   └── pkg/                    # 内部工具包
│       ├── errors/             # 错误处理
│       ├── logger/             # 日志工具
│       └── utils/              # 通用工具
│
├── pkg/                        # 可导出的公共包
│   └── client/                 # SDK 客户端 (可选)
│
├── api/                        # API 定义
│   └── openapi.yaml            # OpenAPI 规范
│
├── configs/                    # 配置文件
│   ├── config.yaml             # 默认配置
│   ├── config.dev.yaml         # 开发环境配置
│   └── config.prod.yaml        # 生产环境配置
│
├── scripts/                    # 脚本文件
│   ├── build.sh                # 构建脚本
│   └── docker-entrypoint.sh    # Docker 入口脚本
│
├── deployments/                # 部署配置
│   ├── Dockerfile              # Docker 镜像
│   └── docker-compose.yaml     # 本地开发环境
│
├── docs/                       # 文档
│   └── api.md                  # API 文档
│
├── test/                       # 集成测试
│   └── integration/
│
├── go.mod                      # Go 模块定义
├── go.sum                      # 依赖锁定
├── main.go                     # 程序入口
├── Makefile                    # 构建命令
└── README.md                   # 项目说明
```

---

## 4. 核心模块设计

### 4.1 CLI 层 (Cobra + Viper 深度集成)

#### 4.1.1 设计原则

配置优先级 (从高到低):
1. **命令行参数** (--port, --config)
2. **环境变量** (LEMON_SERVER_PORT)
3. **配置文件** (config.yaml)
4. **默认值** (代码中定义)

#### 4.1.2 根命令 (root.go)

```go
// cmd/root.go
package cmd

import (
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"

    "lemon/internal/config"
)

var (
    cfgFile string
    cfg     *config.Config
)

var rootCmd = &cobra.Command{
    Use:   "lemon",
    Short: "Lemon - AI-powered API Service",
    Long: `Lemon is an AI-powered API service built with Eino framework.
It provides LLM chat, intelligent agent, and more AI capabilities.`,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // 在所有子命令执行前加载配置
        return initConfig()
    },
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // 全局持久化 flags (所有子命令可用)
    rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
        "config file (default: ./configs/config.yaml)")

    // 绑定到 viper
    viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}

func initConfig() error {
    if cfgFile != "" {
        // 使用指定的配置文件
        viper.SetConfigFile(cfgFile)
    } else {
        // 默认配置文件搜索路径
        viper.SetConfigName("config")
        viper.SetConfigType("yaml")
        viper.AddConfigPath("./configs")
        viper.AddConfigPath(".")
        viper.AddConfigPath("$HOME/.lemon")
    }

    // 环境变量设置
    viper.SetEnvPrefix("LEMON")                 // 前缀: LEMON_
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // server.port -> SERVER_PORT
    viper.AutomaticEnv()                        // 自动绑定环境变量

    // 读取配置文件
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // 配置文件不存在，使用默认值
            fmt.Println("No config file found, using defaults and environment variables")
        } else {
            return fmt.Errorf("failed to read config: %w", err)
        }
    } else {
        fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
    }

    // 反序列化到结构体
    cfg = &config.Config{}
    if err := viper.Unmarshal(cfg); err != nil {
        return fmt.Errorf("failed to unmarshal config: %w", err)
    }

    return nil
}

// GetConfig 返回全局配置
func GetConfig() *config.Config {
    return cfg
}
```

#### 4.1.3 Serve 命令 (serve.go)

```go
// cmd/serve.go
package cmd

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"

    "lemon/internal/server"
)

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the API server",
    Long:  `Start the Lemon API server with the specified configuration.`,
    RunE:  runServe,
}

func init() {
    rootCmd.AddCommand(serveCmd)

    // 定义 flags
    flags := serveCmd.Flags()

    // 服务器配置
    flags.StringP("host", "H", "0.0.0.0", "server host")
    flags.IntP("port", "p", 8080, "server port")
    flags.String("mode", "release", "server mode (debug/release/test)")

    // AI 配置
    flags.String("ai-provider", "openai", "AI provider (openai/azure/anthropic)")
    flags.String("ai-model", "gpt-4", "AI model name")
    flags.String("ai-api-key", "", "AI API key (recommend using env: LEMON_AI_API_KEY)")

    // 日志配置
    flags.String("log-level", "info", "log level (debug/info/warn/error)")
    flags.String("log-format", "json", "log format (json/text)")

    // 绑定 flags 到 viper (使用嵌套路径)
    viper.BindPFlag("server.host", flags.Lookup("host"))
    viper.BindPFlag("server.port", flags.Lookup("port"))
    viper.BindPFlag("server.mode", flags.Lookup("mode"))
    viper.BindPFlag("ai.provider", flags.Lookup("ai-provider"))
    viper.BindPFlag("ai.model", flags.Lookup("ai-model"))
    viper.BindPFlag("ai.api_key", flags.Lookup("ai-api-key"))
    viper.BindPFlag("log.level", flags.Lookup("log-level"))
    viper.BindPFlag("log.format", flags.Lookup("log-format"))

    // 设置默认值
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("server.mode", "release")
    viper.SetDefault("ai.provider", "openai")
    viper.SetDefault("ai.model", "gpt-4")
    viper.SetDefault("log.level", "info")
    viper.SetDefault("log.format", "json")
}

func runServe(cmd *cobra.Command, args []string) error {
    cfg := GetConfig()

    // 打印配置信息 (调试用)
    if cfg.Server.Mode == "debug" {
        fmt.Printf("Server Config: %+v\n", cfg.Server)
        fmt.Printf("AI Config: provider=%s, model=%s\n", cfg.AI.Provider, cfg.AI.Model)
    }

    // 创建服务器
    srv, err := server.New(cfg)
    if err != nil {
        return fmt.Errorf("failed to create server: %w", err)
    }

    // 优雅关闭
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 监听系统信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("\nShutting down gracefully...")
        cancel()
    }()

    // 启动服务器
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    fmt.Printf("Starting server on %s\n", addr)

    return srv.Run(ctx, addr)
}
```

#### 4.1.4 Version 命令 (version.go)

```go
// cmd/version.go
package cmd

import (
    "fmt"
    "runtime"

    "github.com/spf13/cobra"
)

var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("Lemon %s\n", Version)
        fmt.Printf("  Git Commit: %s\n", GitCommit)
        fmt.Printf("  Build Time: %s\n", BuildTime)
        fmt.Printf("  Go Version: %s\n", runtime.Version())
        fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
    },
}

func init() {
    rootCmd.AddCommand(versionCmd)
}
```

#### 4.1.5 配置结构体 (config.go)

```go
// internal/config/config.go
package config

import "time"

// Config 应用配置根结构
type Config struct {
    Server ServerConfig `mapstructure:"server"`
    AI     AIConfig     `mapstructure:"ai"`
    Log    LogConfig    `mapstructure:"log"`
    Mongo  MongoConfig  `mapstructure:"mongo"`
    Redis  RedisConfig  `mapstructure:"redis"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
    Host         string        `mapstructure:"host"`
    Port         int           `mapstructure:"port"`
    Mode         string        `mapstructure:"mode"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// AIConfig AI 服务配置
type AIConfig struct {
    Provider string          `mapstructure:"provider"`
    APIKey   string          `mapstructure:"api_key"`
    Model    string          `mapstructure:"model"`
    BaseURL  string          `mapstructure:"base_url"`
    Options  AIOptionsConfig `mapstructure:"options"`
}

// AIOptionsConfig AI 模型参数
type AIOptionsConfig struct {
    Temperature float64 `mapstructure:"temperature"`
    MaxTokens   int     `mapstructure:"max_tokens"`
    TopP        float64 `mapstructure:"top_p"`
}

// LogConfig 日志配置 (Zerolog)
type LogConfig struct {
    Level      string `mapstructure:"level"`       // trace, debug, info, warn, error, fatal
    Format     string `mapstructure:"format"`      // json, console
    Output     string `mapstructure:"output"`      // stdout, file
    FilePath   string `mapstructure:"file_path"`
    TimeFormat string `mapstructure:"time_format"` // RFC3339, Unix, UnixMs
}

// MongoConfig MongoDB 配置
type MongoConfig struct {
    URI         string `mapstructure:"uri"`          // mongodb://localhost:27017
    Database    string `mapstructure:"database"`     // 数据库名
    MaxPoolSize uint64 `mapstructure:"max_pool_size"`
    MinPoolSize uint64 `mapstructure:"min_pool_size"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
    Addr     string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
    if c.AI.APIKey == "" {
        return fmt.Errorf("AI API key is required")
    }
    if c.Server.Port <= 0 || c.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", c.Server.Port)
    }
    return nil
}
```

#### 4.1.6 使用示例

```bash
# 使用默认配置
lemon serve

# 指定配置文件
lemon serve -c /path/to/config.yaml

# 命令行参数覆盖
lemon serve --port 9090 --log-level debug

# 环境变量覆盖 (优先级: flag > env > config file)
export LEMON_SERVER_PORT=9090
export LEMON_AI_API_KEY=sk-xxx
export LEMON_LOG_LEVEL=debug
lemon serve

# 组合使用
LEMON_AI_API_KEY=sk-xxx lemon serve -c config.prod.yaml --port 8080
```

#### 4.1.7 配置优先级示意图

```
┌─────────────────────────────────────────────────────────┐
│                    配置优先级                            │
├─────────────────────────────────────────────────────────┤
│  1. 命令行参数 (--port 9090)                   最高优先  │
│         ↓                                               │
│  2. 环境变量 (LEMON_SERVER_PORT=9090)                   │
│         ↓                                               │
│  3. 配置文件 (config.yaml: server.port: 8080)           │
│         ↓                                               │
│  4. 默认值 (viper.SetDefault)                  最低优先  │
└─────────────────────────────────────────────────────────┘
```

#### 4.1.8 环境变量映射规则

| 配置路径 | 环境变量 | 命令行参数 |
|----------|----------|------------|
| server.host | LEMON_SERVER_HOST | --host |
| server.port | LEMON_SERVER_PORT | --port |
| ai.api_key | LEMON_AI_API_KEY | --ai-api-key |
| ai.provider | LEMON_AI_PROVIDER | --ai-provider |
| log.level | LEMON_LOG_LEVEL | --log-level |

#### 4.1.9 配置热更新 (可选)

```go
// cmd/root.go 添加配置热更新支持
func initConfigWithWatch() error {
    if err := initConfig(); err != nil {
        return err
    }

    // 监听配置文件变化
    viper.WatchConfig()
    viper.OnConfigChange(func(e fsnotify.Event) {
        fmt.Printf("Config file changed: %s\n", e.Name)
        // 重新加载配置
        if err := viper.Unmarshal(cfg); err != nil {
            fmt.Printf("Failed to reload config: %v\n", err)
        }
        // 通知其他组件配置已更新
        // 注意: 某些配置 (如端口) 可能需要重启才能生效
    })

    return nil
}
```

### 4.2 AI 层 (Eino 集成)

#### 4.2.1 模型组件封装

```go
// internal/ai/component/model.go
package component

import (
    "context"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

type ModelConfig struct {
    Provider string
    APIKey   string
    Model    string
    BaseURL  string
}

func NewChatModel(cfg ModelConfig) (model.ChatModel, error) {
    switch cfg.Provider {
    case "openai":
        return openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
            APIKey:  cfg.APIKey,
            Model:   cfg.Model,
            BaseURL: cfg.BaseURL,
        })
    // 支持其他提供商...
    default:
        return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
    }
}
```

#### 4.2.2 对话 Chain

```go
// internal/ai/chain/chat_chain.go
package chain

import (
    "context"

    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

type ChatChain struct {
    chain compose.Runnable[string, *schema.Message]
}

func NewChatChain(model model.ChatModel) (*ChatChain, error) {
    // 构建对话链
    chain, err := compose.NewChain[string, *schema.Message]().
        AppendChatTemplate(promptTemplate).
        AppendChatModel(model).
        Compile(context.Background())
    if err != nil {
        return nil, err
    }

    return &ChatChain{chain: chain}, nil
}

func (c *ChatChain) Run(ctx context.Context, input string) (*schema.Message, error) {
    return c.chain.Invoke(ctx, input)
}

func (c *ChatChain) Stream(ctx context.Context, input string) (<-chan *schema.Message, error) {
    return c.chain.Stream(ctx, input)
}
```

#### 4.2.3 智能代理 Graph

```go
// internal/ai/graph/agent_graph.go
package graph

import (
    "context"

    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/flow/agent/react"
)

type AgentGraph struct {
    agent *react.Agent
}

func NewAgentGraph(model model.ChatModel, tools []tool.BaseTool) (*AgentGraph, error) {
    agent, err := react.NewAgent(context.Background(), &react.AgentConfig{
        Model: model,
        Tools: tools,
    })
    if err != nil {
        return nil, err
    }

    return &AgentGraph{agent: agent}, nil
}

func (a *AgentGraph) Run(ctx context.Context, input string) (*schema.Message, error) {
    return a.agent.Generate(ctx, []*schema.Message{
        schema.UserMessage(input),
    })
}
```

### 4.3 Service 层

```go
// internal/service/chat_service.go
package service

import (
    "context"

    "lemon/internal/ai/chain"
    "lemon/internal/model"
)

type ChatService struct {
    chatChain *chain.ChatChain
}

func NewChatService(chatChain *chain.ChatChain) *ChatService {
    return &ChatService{chatChain: chatChain}
}

func (s *ChatService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
    msg, err := s.chatChain.Run(ctx, req.Message)
    if err != nil {
        return nil, err
    }

    return &model.ChatResponse{
        Message: msg.Content,
    }, nil
}

func (s *ChatService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan *model.ChatChunk, error) {
    // 流式响应实现
}
```

### 4.4 Handler 层

```go
// internal/handler/chat.go
package handler

import (
    "github.com/gin-gonic/gin"

    "lemon/internal/model"
    "lemon/internal/service"
)

type ChatHandler struct {
    chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
    return &ChatHandler{chatService: chatService}
}

func (h *ChatHandler) Chat(c *gin.Context) {
    var req model.ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.chatService.Chat(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, resp)
}

func (h *ChatHandler) ChatStream(c *gin.Context) {
    // SSE 流式响应实现
}
```

---

## 5. API 设计

### 5.1 接口列表

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /health | 健康检查 |
| POST | /api/v1/chat | 对话接口 |
| POST | /api/v1/chat/stream | 流式对话接口 (SSE) |
| POST | /api/v1/agent | 智能代理接口 |
| POST | /api/v1/agent/stream | 流式代理接口 (SSE) |

### 5.2 请求/响应示例

#### 对话接口

**请求**:
```json
POST /api/v1/chat
{
    "message": "你好，请介绍一下自己",
    "conversation_id": "conv_123",
    "options": {
        "temperature": 0.7,
        "max_tokens": 2048
    }
}
```

**响应**:
```json
{
    "message": "你好！我是 Lemon AI 助手...",
    "conversation_id": "conv_123",
    "usage": {
        "prompt_tokens": 10,
        "completion_tokens": 50,
        "total_tokens": 60
    }
}
```

#### 流式对话接口

**请求**:
```json
POST /api/v1/chat/stream
{
    "message": "写一首诗"
}
```

**响应** (SSE):
```
event: message
data: {"content": "春"}

event: message
data: {"content": "风"}

event: done
data: {"usage": {"total_tokens": 100}}
```

---

## 6. 配置管理

### 6.1 配置结构

```yaml
# configs/config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"           # debug, release, test
  read_timeout: 30s
  write_timeout: 30s

ai:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4"
  base_url: "https://api.openai.com/v1"
  options:
    temperature: 0.7
    max_tokens: 4096
    top_p: 1.0

log:
  level: "info"             # trace, debug, info, warn, error, fatal
  format: "json"            # json, console
  output: "stdout"          # stdout, file
  file_path: "./logs/lemon.log"
  time_format: "RFC3339"    # RFC3339, Unix, UnixMs

mongo:
  uri: "mongodb://localhost:27017"
  database: "lemon"
  max_pool_size: 100
  min_pool_size: 10

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
```

### 6.2 配置加载

配置结构体定义参见 [4.1.5 配置结构体](#415-配置结构体-configgo)

---

## 7. 数据存储 (MongoDB + Redis)

### 7.1 MongoDB 客户端初始化

```go
// internal/pkg/mongodb/client.go
package mongodb

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"

    "lemon/internal/config"
)

type Client struct {
    client   *mongo.Client
    database *mongo.Database
}

func New(cfg *config.MongoConfig) (*Client, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 配置客户端选项
    clientOpts := options.Client().
        ApplyURI(cfg.URI).
        SetMaxPoolSize(cfg.MaxPoolSize).
        SetMinPoolSize(cfg.MinPoolSize)

    // 连接 MongoDB
    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        return nil, err
    }

    // 验证连接
    if err := client.Ping(ctx, readpref.Primary()); err != nil {
        return nil, err
    }

    return &Client{
        client:   client,
        database: client.Database(cfg.Database),
    }, nil
}

func (c *Client) Database() *mongo.Database {
    return c.database
}

func (c *Client) Collection(name string) *mongo.Collection {
    return c.database.Collection(name)
}

func (c *Client) Close(ctx context.Context) error {
    return c.client.Disconnect(ctx)
}
```

### 7.2 数据模型定义

```go
// internal/model/entity.go
package model

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation 对话实体
type Conversation struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID    string             `bson:"user_id" json:"user_id"`
    Title     string             `bson:"title" json:"title"`
    Model     string             `bson:"model" json:"model"`
    Messages  []Message          `bson:"messages" json:"messages"`
    Metadata  map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"`
    CreatedAt time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Message 消息
type Message struct {
    Role      string    `bson:"role" json:"role"`           // user, assistant, system
    Content   string    `bson:"content" json:"content"`
    Timestamp time.Time `bson:"timestamp" json:"timestamp"`
    TokenUsage *TokenUsage `bson:"token_usage,omitempty" json:"token_usage,omitempty"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
    PromptTokens     int `bson:"prompt_tokens" json:"prompt_tokens"`
    CompletionTokens int `bson:"completion_tokens" json:"completion_tokens"`
    TotalTokens      int `bson:"total_tokens" json:"total_tokens"`
}
```

### 7.3 Repository 层

```go
// internal/repository/conversation_repo.go
package repository

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "lemon/internal/model"
)

type ConversationRepo struct {
    collection *mongo.Collection
}

func NewConversationRepo(db *mongo.Database) *ConversationRepo {
    return &ConversationRepo{
        collection: db.Collection("conversations"),
    }
}

// Create 创建对话
func (r *ConversationRepo) Create(ctx context.Context, conv *model.Conversation) error {
    conv.CreatedAt = time.Now()
    conv.UpdatedAt = time.Now()

    result, err := r.collection.InsertOne(ctx, conv)
    if err != nil {
        return err
    }

    conv.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}

// FindByID 根据 ID 查询
func (r *ConversationRepo) FindByID(ctx context.Context, id string) (*model.Conversation, error) {
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, err
    }

    var conv model.Conversation
    err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&conv)
    if err != nil {
        return nil, err
    }

    return &conv, nil
}

// AppendMessage 追加消息
func (r *ConversationRepo) AppendMessage(ctx context.Context, id string, msg model.Message) error {
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return err
    }

    update := bson.M{
        "$push": bson.M{"messages": msg},
        "$set":  bson.M{"updated_at": time.Now()},
    }

    _, err = r.collection.UpdateByID(ctx, objectID, update)
    return err
}

// ListByUserID 查询用户对话列表
func (r *ConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int64) ([]*model.Conversation, error) {
    opts := options.Find().
        SetSort(bson.D{{Key: "updated_at", Value: -1}}).
        SetLimit(limit).
        SetSkip(offset).
        SetProjection(bson.M{"messages": 0}) // 不返回消息详情

    cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var convs []*model.Conversation
    if err := cursor.All(ctx, &convs); err != nil {
        return nil, err
    }

    return convs, nil
}
```

### 7.4 Redis 缓存集成

```go
// internal/pkg/cache/redis.go
package cache

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"

    "lemon/internal/config"
)

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(cfg *config.RedisConfig) (*RedisCache, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr,
        Password: cfg.Password,
        DB:       cfg.DB,
    })

    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, err
    }

    return &RedisCache{client: client}, nil
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string, dest any) error {
    data, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return err
    }
    return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
    return c.client.Del(ctx, keys...).Err()
}

// 会话缓存 key 示例
const (
    ConversationCacheKeyPrefix = "conv:"
    ConversationCacheTTL       = 30 * time.Minute
)

func ConversationCacheKey(id string) string {
    return ConversationCacheKeyPrefix + id
}
```

### 7.5 索引设计

```go
// internal/pkg/mongodb/indexes.go
package mongodb

import (
    "context"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes 创建索引
func EnsureIndexes(db *mongo.Database) error {
    ctx := context.Background()

    // conversations 集合索引
    convColl := db.Collection("conversations")
    convIndexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "updated_at", Value: -1}},
            Options: options.Index().SetName("idx_user_updated"),
        },
        {
            Keys:    bson.D{{Key: "created_at", Value: -1}},
            Options: options.Index().SetName("idx_created").SetExpireAfterSeconds(86400 * 90), // 90天TTL
        },
    }

    _, err := convColl.Indexes().CreateMany(ctx, convIndexes)
    return err
}
```

---

## 8. 错误处理

### 8.1 错误码定义

```go
// internal/pkg/errors/errors.go
package errors

type ErrorCode int

const (
    ErrCodeUnknown       ErrorCode = 10000
    ErrCodeInvalidParam  ErrorCode = 10001
    ErrCodeUnauthorized  ErrorCode = 10002
    ErrCodeForbidden     ErrorCode = 10003
    ErrCodeNotFound      ErrorCode = 10004

    ErrCodeAIError       ErrorCode = 20001
    ErrCodeModelTimeout  ErrorCode = 20002
    ErrCodeRateLimited   ErrorCode = 20003
)

type AppError struct {
    Code    ErrorCode `json:"code"`
    Message string    `json:"message"`
    Detail  string    `json:"detail,omitempty"`
}

func (e *AppError) Error() string {
    return e.Message
}
```

### 8.2 统一错误响应

```json
{
    "code": 20001,
    "message": "AI service error",
    "detail": "model response timeout"
}
```

---

## 8. 日志与监控

### 8.1 Zerolog 初始化

```go
// internal/pkg/logger/logger.go
package logger

import (
    "io"
    "os"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"

    "lemon/internal/config"
)

// Init 初始化全局日志
func Init(cfg *config.LogConfig) error {
    // 设置日志级别
    level, err := zerolog.ParseLevel(cfg.Level)
    if err != nil {
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)

    // 设置时间格式
    switch cfg.TimeFormat {
    case "Unix":
        zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    case "UnixMs":
        zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
    default:
        zerolog.TimeFieldFormat = time.RFC3339
    }

    // 设置输出
    var output io.Writer = os.Stdout
    if cfg.Output == "file" && cfg.FilePath != "" {
        file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return err
        }
        output = file
    }

    // Console 格式 (开发环境友好)
    if cfg.Format == "console" {
        output = zerolog.ConsoleWriter{
            Out:        output,
            TimeFormat: time.RFC3339,
        }
    }

    // 设置全局 logger
    log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

    return nil
}

// Get 获取带上下文的 logger
func Get() zerolog.Logger {
    return log.Logger
}
```

### 8.2 日志使用示例

```go
// 结构化日志示例 (Zerolog)
import "github.com/rs/zerolog/log"

// 基本日志
log.Info().
    Str("conversation_id", convID).
    Int("prompt_tokens", usage.PromptTokens).
    Dur("latency", latency).
    Msg("chat request processed")

// 错误日志
log.Error().
    Err(err).
    Str("user_id", userID).
    Msg("failed to process chat request")

// 带请求上下文的日志
func (h *ChatHandler) Chat(c *gin.Context) {
    logger := log.With().
        Str("request_id", c.GetHeader("X-Request-ID")).
        Str("client_ip", c.ClientIP()).
        Logger()

    logger.Info().Msg("handling chat request")
    // ...
}

// 子 logger (性能优化，避免重复字段)
chatLogger := log.With().Str("module", "chat").Logger()
chatLogger.Info().Msg("chat service started")
```

### 8.3 Gin 日志中间件

```go
// internal/server/middleware/logging.go
package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
)

func ZerologMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery

        c.Next()

        latency := time.Since(start)
        status := c.Writer.Status()

        event := log.Info()
        if status >= 400 {
            event = log.Warn()
        }
        if status >= 500 {
            event = log.Error()
        }

        event.
            Str("method", c.Request.Method).
            Str("path", path).
            Str("query", query).
            Int("status", status).
            Dur("latency", latency).
            Str("client_ip", c.ClientIP()).
            Str("user_agent", c.Request.UserAgent()).
            Msg("HTTP request")
    }
}
```

### 8.4 监控指标

| 指标名 | 类型 | 描述 |
|--------|------|------|
| http_requests_total | Counter | HTTP 请求总数 |
| http_request_duration_seconds | Histogram | 请求耗时 |
| ai_model_requests_total | Counter | 模型调用次数 |
| ai_model_tokens_total | Counter | Token 消耗总量 |
| ai_model_latency_seconds | Histogram | 模型响应延迟 |

### 8.5 Eino Callback 集成 (Zerolog)

```go
// internal/ai/callback/logging.go
package callback

import (
    "context"

    "github.com/cloudwego/eino/callbacks"
    "github.com/rs/zerolog"
)

type LoggingCallback struct {
    logger zerolog.Logger
}

func NewLoggingCallback(logger zerolog.Logger) *LoggingCallback {
    return &LoggingCallback{
        logger: logger.With().Str("module", "eino").Logger(),
    }
}

func (c *LoggingCallback) OnStart(ctx context.Context, info *callbacks.RunInfo) context.Context {
    c.logger.Debug().
        Str("component", info.Component).
        Str("type", info.Type).
        Msg("component started")
    return ctx
}

func (c *LoggingCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo) context.Context {
    c.logger.Debug().
        Str("component", info.Component).
        Dur("duration", info.Duration).
        Msg("component finished")
    return ctx
}

func (c *LoggingCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
    c.logger.Error().
        Err(err).
        Str("component", info.Component).
        Msg("component error")
    return ctx
}
```

---

## 9. 依赖注入与初始化

### 9.1 应用启动流程

```
main.go
    └── cmd.Execute()
        └── serveCmd.RunE()
            ├── config.Load()           # 1. 加载配置
            ├── logger.Init()           # 2. 初始化日志
            ├── ai.Init()               # 3. 初始化 AI 组件
            │   ├── component.NewModel()
            │   ├── chain.NewChatChain()
            │   └── graph.NewAgentGraph()
            ├── service.Init()          # 4. 初始化服务层
            ├── handler.Init()          # 5. 初始化处理器
            ├── server.Init()           # 6. 初始化服务器
            │   ├── router.Setup()
            │   └── middleware.Setup()
            └── server.Run()            # 7. 启动服务
```

### 9.2 依赖注入示例 (Wire 可选)

```go
// internal/wire.go
// +build wireinject

package internal

import (
    "github.com/google/wire"
)

func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(
        // AI 组件
        component.NewChatModel,
        chain.NewChatChain,
        graph.NewAgentGraph,

        // 服务层
        service.NewChatService,
        service.NewAgentService,

        // 处理器
        handler.NewChatHandler,
        handler.NewAgentHandler,

        // 服务器
        server.NewServer,

        // App
        NewApp,
    )
    return nil, nil
}
```

---

## 10. 部署方案

### 10.1 Dockerfile

```dockerfile
# deployments/Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lemon .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/lemon .
COPY configs/config.yaml ./configs/

EXPOSE 8080
ENTRYPOINT ["./lemon"]
CMD ["serve"]
```

### 10.2 Docker Compose (开发环境)

```yaml
# deployments/docker-compose.yaml
version: '3.8'

services:
  lemon:
    build:
      context: ..
      dockerfile: deployments/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - LEMON_AI_API_KEY=${OPENAI_API_KEY}
      - LEMON_MONGO_URI=mongodb://mongo:27017
      - LEMON_REDIS_ADDR=redis:6379
    volumes:
      - ../configs:/app/configs
    depends_on:
      - mongo
      - redis

  mongo:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db
    environment:
      - MONGO_INITDB_DATABASE=lemon

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  mongo_data:
  redis_data:
```

### 10.3 Makefile

```makefile
.PHONY: build run test lint

build:
	go build -o bin/lemon .

run:
	go run . serve

test:
	go test -v ./...

lint:
	golangci-lint run

docker-build:
	docker build -t lemon:latest -f deployments/Dockerfile .

docker-run:
	docker-compose -f deployments/docker-compose.yaml up -d
```

---

## 11. 开发规范

### 11.1 代码规范

- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 使用 `golangci-lint` 进行代码检查
- 错误处理: 不要忽略错误，适当包装错误信息
- 注释: 公开 API 必须有文档注释

### 11.2 Git 规范

```
feat: 新功能
fix: Bug 修复
docs: 文档更新
refactor: 重构
test: 测试
chore: 构建/工具
```

### 11.3 测试规范

- 单元测试: `*_test.go`
- 集成测试: `test/integration/`
- 覆盖率目标: >= 70%

---

## 12. 后续扩展

### 12.1 可选功能

- [ ] RAG 检索增强生成
- [ ] 多模态支持 (图片、语音)
- [ ] 会话历史持久化
- [ ] 用户认证 (JWT/OAuth)
- [ ] 速率限制
- [ ] A/B 测试
- [ ] 模型切换与降级

### 12.2 扩展点

1. **新增 AI Provider**: 在 `internal/ai/component/model.go` 添加
2. **新增工具**: 在 `internal/ai/component/tool.go` 实现 `tool.BaseTool` 接口
3. **新增 API**: 在 `internal/handler/` 添加处理器

---

## 附录

### A. 参考资料

- [Eino 官方文档](https://github.com/cloudwego/eino)
- [Cobra 文档](https://cobra.dev/)
- [Viper 文档](https://github.com/spf13/viper)

### B. 依赖列表

```
# 核心框架
github.com/spf13/cobra          # CLI 框架
github.com/spf13/viper          # 配置管理
github.com/gin-gonic/gin        # HTTP 框架

# AI
github.com/cloudwego/eino       # Eino 核心
github.com/cloudwego/eino-ext   # Eino 扩展组件

# 日志
github.com/rs/zerolog           # 零分配 JSON 日志

# 数据存储
go.mongodb.org/mongo-driver     # MongoDB 驱动
github.com/redis/go-redis/v9    # Redis 客户端

# 工具
github.com/google/wire          # 依赖注入 (可选)
github.com/fsnotify/fsnotify    # 配置热更新 (可选)
```
