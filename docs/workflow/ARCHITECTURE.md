# AI视频生成工作流系统架构设计文档

## 1. 系统概述

### 1.1 项目背景

AI视频生成工作流系统是一个基于Lemon项目的扩展模块，提供从原始素材（小说、文档、图片等）到最终视频的完整自动化生成流程。系统通过编排多个AI服务，实现剧本生成、资产设计、分镜生成、视频制作等全流程自动化。

### 1.2 核心特性

- **全流程自动化**: 从输入到输出的端到端自动化处理
- **多AI服务集成**: 支持Gemini3 Pro、豆包、Seedream、Sora2等多种AI服务
- **异步任务处理**: 基于消息队列的异步任务处理，支持长时间运行的任务
- **状态持久化**: 完整的工作流状态管理，支持暂停、恢复、重试
- **文件存储**: 支持S3对象存储和本地文件系统，处理大文件存储

### 1.3 工作流阶段

1. **剧本生成**: 小说/文档/图片 → 剧本 → 分镜头脚本
2. **资产设计**: 提取人物/道具/场景，生成设计图和提示词
3. **分镜画面生成**: 根据分镜脚本生成分镜图，支持网格排版和编辑
4. **动态分镜粗剪**: 整合分镜图和台词，生成动态分镜和音频
5. **视频生成**: 基于动态分镜生成视频片段
6. **视频剪辑**: 整合视频片段，添加音效和背景音乐

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│                    (HTTP REST API)                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      Handler Layer                           │
│  workflow.go | script.go | asset.go | storyboard.go | ...   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  WorkflowService (编排)                                      │
│    ├── ScriptService (剧本生成)                              │
│    ├── AssetService (资产设计)                               │
│    ├── StoryboardService (分镜生成)                          │
│    ├── AnimaticService (动态分镜)                            │
│    ├── VideoService (视频生成)                               │
│    └── EditService (视频剪辑)                                │
└─────────────────────────────────────────────────────────────┘
         ↓                    ↓                    ↓
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  AI Clients  │    │   Storage    │    │    Queue     │
│              │    │              │    │              │
│ Gemini3 Pro  │    │  S3/MinIO    │    │ Redis Streams│
│ 豆包         │    │  Local FS    │    │ /RabbitMQ    │
│ Seedream     │    │              │    │              │
│ ...          │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
         ↓                    ↓                    ↓
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                      │
│  MongoDB (状态持久化) | Redis (缓存) | Worker (任务处理)    │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 分层设计

#### 2.2.1 Handler层 (接口适配层)

**职责**:
- HTTP请求解析和参数校验
- 响应封装和错误处理
- 请求路由

**文件结构**:
```
internal/handler/
├── workflow.go      # 工作流管理接口
├── script.go        # 剧本生成接口
├── asset.go         # 资产设计接口
├── storyboard.go    # 分镜生成接口
├── animatic.go      # 动态分镜接口
├── video.go         # 视频生成接口
└── edit.go          # 视频剪辑接口
```

#### 2.2.2 Service层 (业务逻辑层)

**职责**:
- 业务流程编排
- 业务规则校验
- 事务管理
- 调用AI服务和存储服务

**文件结构**:
```
internal/service/
├── workflow_service.go    # 工作流编排服务
├── script_service.go     # 剧本生成服务
├── asset_service.go      # 资产设计服务
├── storyboard_service.go  # 分镜生成服务
├── animatic_service.go    # 动态分镜服务
├── video_service.go       # 视频生成服务
├── edit_service.go        # 视频剪辑服务
├── document_parser.go     # 文档解析服务
└── audio_service.go       # 音频生成服务
```

#### 2.2.3 AI层 (AI能力层)

**职责**:
- 封装各AI服务的客户端
- 统一的AI服务接口
- 错误处理和重试

**文件结构**:
```
internal/ai/client/
├── base.go         # AI客户端基础接口
├── gemini.go       # Gemini3 Pro客户端
├── doubao.go       # 豆包客户端
├── seedream.go     # Seedream客户端
├── sora2.go        # Sora2客户端
├── minimax.go      # Minimax客户端
├── elevenlabs.go   # ElevenLabs客户端
└── ...             # 其他AI服务客户端
```

#### 2.2.4 Infrastructure层 (基础设施层)

**职责**:
- 任务队列管理
- 文件存储管理
- 数据持久化

**文件结构**:
```
internal/pkg/
├── queue/
│   └── queue.go          # 任务队列接口和实现
├── storage/
│   ├── storage.go        # 存储接口
│   ├── s3.go             # S3存储实现
│   └── local.go          # 本地存储实现
└── image/
    └── processor.go      # 图片处理工具

internal/worker/
├── task_processor.go     # 任务处理器
└── tasks.go             # 任务定义
```

## 3. 数据模型设计

### 3.1 核心实体

#### 3.1.1 Workflow (工作流)

```go
type Workflow struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID      string             `bson:"user_id" json:"user_id"`
    Name        string             `bson:"name" json:"name"`
    Status      WorkflowStatus     `bson:"status" json:"status"`
    CurrentStage WorkflowStage     `bson:"current_stage" json:"current_stage"`
    
    // 各阶段数据
    Script      *Script            `bson:"script,omitempty" json:"script,omitempty"`
    Assets      []*Asset           `bson:"assets,omitempty" json:"assets,omitempty"`
    Storyboard  *Storyboard        `bson:"storyboard,omitempty" json:"storyboard,omitempty"`
    Animatic    *Animatic          `bson:"animatic,omitempty" json:"animatic,omitempty"`
    Video       *Video             `bson:"video,omitempty" json:"video,omitempty"`
    Edit        *Edit              `bson:"edit,omitempty" json:"edit,omitempty"`
    
    // 元数据
    Metadata    map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"`
    Error       string             `bson:"error,omitempty" json:"error,omitempty"`
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
    CompletedAt *time.Time         `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}
```

#### 3.1.2 Script (剧本)

```go
type Script struct {
    ID              string    `bson:"id" json:"id"`
    InputType       string    `bson:"input_type" json:"input_type"` // novel, document, image, text
    InputContent    string    `bson:"input_content" json:"input_content"`
    InputFileURL    string    `bson:"input_file_url,omitempty" json:"input_file_url,omitempty"`
    Novel           string    `bson:"novel,omitempty" json:"novel,omitempty"`
    Screenplay      string    `bson:"screenplay" json:"screenplay"`
    StoryboardScript string   `bson:"storyboard_script" json:"storyboard_script"`
    Status          string    `bson:"status" json:"status"`
    CreatedAt       time.Time `bson:"created_at" json:"created_at"`
}
```

#### 3.1.3 Asset (资产)

```go
type Asset struct {
    ID          string    `bson:"id" json:"id"`
    Type        string    `bson:"type" json:"type"` // character, prop, scene
    Name        string    `bson:"name" json:"name"`
    Description string    `bson:"description" json:"description"`
    Prompt      string    `bson:"prompt" json:"prompt"`
    ReferenceImageURL string `bson:"reference_image_url,omitempty" json:"reference_image_url,omitempty"`
    DesignImageURL string   `bson:"design_image_url,omitempty" json:"design_image_url,omitempty"`
    Status      string    `bson:"status" json:"status"`
    CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}
```

#### 3.1.4 Storyboard (分镜)

```go
type Storyboard struct {
    ID          string    `bson:"id" json:"id"`
    Shots       []*Shot   `bson:"shots" json:"shots"`
    GridLayout  string    `bson:"grid_layout,omitempty" json:"grid_layout,omitempty"` // 3x3, 5x5
    GridImageURL string   `bson:"grid_image_url,omitempty" json:"grid_image_url,omitempty"`
    Status      string    `bson:"status" json:"status"`
    CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}

type Shot struct {
    ID          string    `bson:"id" json:"id"`
    Sequence    int       `bson:"sequence" json:"sequence"`
    Prompt      string    `bson:"prompt" json:"prompt"`
    ShotType    string    `bson:"shot_type" json:"shot_type"` // close-up, medium, wide, etc.
    ImageURL    string    `bson:"image_url" json:"image_url"`
    CroppedImageURL string `bson:"cropped_image_url,omitempty" json:"cropped_image_url,omitempty"`
    Dialogue    string    `bson:"dialogue,omitempty" json:"dialogue,omitempty"`
    Duration    float64   `bson:"duration,omitempty" json:"duration,omitempty"`
}
```

#### 3.1.5 Animatic (动态分镜)

```go
type Animatic struct {
    ID          string         `bson:"id" json:"id"`
    VideoURL    string         `bson:"video_url,omitempty" json:"video_url,omitempty"`
    AudioURL    string         `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
    Shots       []*AnimaticShot `bson:"shots" json:"shots"`
    TotalDuration float64      `bson:"total_duration" json:"total_duration"`
    Status      string         `bson:"status" json:"status"`
    CreatedAt   time.Time      `bson:"created_at" json:"created_at"`
}

type AnimaticShot struct {
    ShotID      string    `bson:"shot_id" json:"shot_id"`
    VideoURL    string    `bson:"video_url,omitempty" json:"video_url,omitempty"`
    AudioURL    string    `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
    StartTime   float64   `bson:"start_time" json:"start_time"`
    Duration    float64   `bson:"duration" json:"duration"`
}
```

#### 3.1.6 Video (视频)

```go
type Video struct {
    ID          string         `bson:"id" json:"id"`
    Segments    []*VideoSegment `bson:"segments" json:"segments"`
    Status      string         `bson:"status" json:"status"`
    Progress    float64        `bson:"progress" json:"progress"`
    CreatedAt   time.Time      `bson:"created_at" json:"created_at"`
}

type VideoSegment struct {
    ID          string    `bson:"id" json:"id"`
    AnimaticShotID string `bson:"animatic_shot_id" json:"animatic_shot_id"`
    VideoURL    string    `bson:"video_url" json:"video_url"`
    Duration    float64   `bson:"duration" json:"duration"`
    Status      string    `bson:"status" json:"status"`
}
```

#### 3.1.7 Edit (剪辑)

```go
type Edit struct {
    ID          string    `bson:"id" json:"id"`
    FinalVideoURL string  `bson:"final_video_url" json:"final_video_url"`
    AudioEffects []string  `bson:"audio_effects,omitempty" json:"audio_effects,omitempty"`
    BackgroundMusicURL string `bson:"background_music_url,omitempty" json:"background_music_url,omitempty"`
    Status      string    `bson:"status" json:"status"`
    CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}
```

### 3.2 状态枚举

```go
type WorkflowStatus string

const (
    WorkflowStatusPending    WorkflowStatus = "pending"
    WorkflowStatusRunning    WorkflowStatus = "running"
    WorkflowStatusPaused     WorkflowStatus = "paused"
    WorkflowStatusCompleted  WorkflowStatus = "completed"
    WorkflowStatusFailed     WorkflowStatus = "failed"
    WorkflowStatusCancelled  WorkflowStatus = "cancelled"
)

type WorkflowStage string

const (
    WorkflowStageScript      WorkflowStage = "script"
    WorkflowStageAsset       WorkflowStage = "asset"
    WorkflowStageStoryboard  WorkflowStage = "storyboard"
    WorkflowStageAnimatic    WorkflowStage = "animatic"
    WorkflowStageVideo       WorkflowStage = "video"
    WorkflowStageEdit        WorkflowStage = "edit"
)
```

## 4. 任务队列设计

### 4.1 任务类型

```go
type TaskType string

const (
    TaskTypeScriptGenerate     TaskType = "script.generate"
    TaskTypeAssetExtract       TaskType = "asset.extract"
    TaskTypeAssetDesign        TaskType = "asset.design"
    TaskTypeStoryboardGenerate TaskType = "storyboard.generate"
    TaskTypeStoryboardCrop     TaskType = "storyboard.crop"
    TaskTypeAnimaticGenerate   TaskType = "animatic.generate"
    TaskTypeAudioGenerate     TaskType = "audio.generate"
    TaskTypeVideoGenerate     TaskType = "video.generate"
    TaskTypeEditFinalize       TaskType = "edit.finalize"
)
```

### 4.2 任务结构

```go
type Task struct {
    ID          string                 `json:"id"`
    Type        TaskType               `json:"type"`
    WorkflowID  string                 `json:"workflow_id"`
    Stage       WorkflowStage           `json:"stage"`
    Payload     map[string]interface{} `json:"payload"`
    RetryCount  int                    `json:"retry_count"`
    MaxRetries  int                    `json:"max_retries"`
    Status      string                 `json:"status"`
    Error       string                 `json:"error,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
}
```

### 4.3 队列实现方案

**方案1: Redis Streams (推荐)**

- 优点: 轻量级，无需额外组件，支持消费者组
- 适用: 中小规模部署

**方案2: RabbitMQ**

- 优点: 功能完善，支持多种消息模式
- 适用: 大规模部署，需要复杂路由

## 5. 文件存储设计

### 5.1 存储接口

```go
type Storage interface {
    // 上传文件
    Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
    
    // 下载文件
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    
    // 获取文件URL
    GetURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
    
    // 删除文件
    Delete(ctx context.Context, key string) error
    
    // 检查文件是否存在
    Exists(ctx context.Context, key string) (bool, error)
}
```

### 5.2 存储路径规范

```
workflows/{workflow_id}/
├── script/
│   ├── input/{filename}
│   └── output/script.txt
├── assets/
│   ├── {asset_id}/
│   │   ├── reference.jpg
│   │   └── design.jpg
├── storyboard/
│   ├── grid/{layout}.jpg
│   └── shots/
│       ├── {shot_id}.jpg
│       └── {shot_id}_cropped.jpg
├── animatic/
│   ├── video.mp4
│   └── audio.mp3
├── video/
│   └── segments/
│       └── {segment_id}.mp4
└── edit/
    └── final.mp4
```

## 6. AI服务集成设计

### 6.1 AI客户端接口

```go
type AIClient interface {
    // 文本生成
    GenerateText(ctx context.Context, prompt string, options *TextOptions) (*TextResponse, error)
    
    // 图片生成
    GenerateImage(ctx context.Context, prompt string, options *ImageOptions) (*ImageResponse, error)
    
    // 图生图
    ImageToImage(ctx context.Context, imageURL string, prompt string, options *ImageOptions) (*ImageResponse, error)
    
    // 视频生成
    GenerateVideo(ctx context.Context, imageURL string, prompt string, options *VideoOptions) (*VideoResponse, error)
    
    // 音频生成
    GenerateAudio(ctx context.Context, text string, options *AudioOptions) (*AudioResponse, error)
}
```

### 6.2 支持的AI服务

| 服务 | 能力 | 用途 |
|------|------|------|
| Gemini3 Pro | 文本生成、多模态 | 剧本生成、资产提取、提示词生成 |
| 豆包 | 文本生成 | 剧本生成、资产提取 |
| Seedream | 图片生成 | 资产设计图生成 |
| 香蕉Pro | 图片生成、图生图 | 分镜图生成、裁剪 |
| 即梦 | 图片生成、视频生成 | 分镜图、视频片段 |
| MJ | 图片生成 | 分镜图生成 |
| Sora2 | 视频生成 | 动态分镜、视频片段 |
| Minimax | 视频生成、音频生成 | 动态分镜、音频 |
| ElevenLabs | 音频生成 | 台词音频 |
| Grok | 视频生成 | 视频片段 |
| Wan | 视频生成 | 视频片段 |
| 可灵 | 视频生成 | 视频片段 |
| 海螺 | 视频生成 | 视频片段 |
| Vidu | 视频生成 | 视频片段 |

## 7. 工作流状态机

### 7.1 状态转换图

```
pending → running → [script] → [asset] → [storyboard] → [animatic] → [video] → [edit] → completed
   ↓         ↓                                                                    ↓
paused   failed/cancelled                                                      failed
   ↓
resume → running
```

### 7.2 状态转换规则

- **pending → running**: 工作流启动
- **running → paused**: 用户暂停或系统暂停
- **paused → running**: 用户恢复
- **running → failed**: 任何阶段失败
- **running → cancelled**: 用户取消
- **running → completed**: 所有阶段完成

## 8. 错误处理和重试

### 8.1 错误分类

- **临时错误**: 网络超时、服务暂时不可用 → 自动重试
- **业务错误**: 参数错误、资源不足 → 记录错误，不重试
- **系统错误**: 服务崩溃、数据损坏 → 告警，人工介入

### 8.2 重试策略

- **最大重试次数**: 3次
- **重试间隔**: 指数退避 (1s, 2s, 4s)
- **重试条件**: 仅对临时错误重试

## 9. 性能优化

### 9.1 并发控制

- 每个工作流阶段独立处理
- 支持并行处理多个工作流
- 限制并发AI服务调用数量

### 9.2 缓存策略

- 工作流状态缓存 (Redis, 5分钟TTL)
- AI服务响应缓存 (可选)
- 文件URL缓存

### 9.3 资源管理

- 大文件流式处理
- 视频文件分片上传
- 定期清理临时文件

## 10. 监控和日志

### 10.1 关键指标

- 工作流创建数量
- 各阶段成功率
- 平均处理时间
- AI服务调用次数和成本
- 存储使用量

### 10.2 日志记录

- 工作流状态变更
- 任务执行日志
- AI服务调用日志
- 错误详情

## 11. 安全考虑

### 11.1 访问控制

- 用户身份验证
- 工作流所有权校验
- API密钥安全管理

### 11.2 数据安全

- 文件访问权限控制
- 敏感数据加密
- 审计日志

## 12. 扩展性设计

### 12.1 水平扩展

- 无状态服务设计
- 任务队列支持多消费者
- 存储服务可扩展

### 12.2 插件化设计

- AI服务客户端可插拔
- 存储后端可替换
- 任务处理器可扩展
