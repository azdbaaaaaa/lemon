# AI视频生成工作流系统 API 设计文档

## 1. API 概览

### 1.1 基础信息

- **Base URL**: `/api/v1/workflow`
- **认证方式**: Bearer Token (JWT)
- **内容类型**: `application/json`
- **文件上传**: `multipart/form-data`

### 1.2 通用响应格式

#### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

#### 错误响应

```json
{
  "code": 40001,
  "message": "Invalid request",
  "detail": "具体错误信息"
}
```

### 1.3 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 202 | 已接受（异步任务） |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

## 2. 工作流管理 API

### 2.1 创建工作流

**POST** `/api/v1/workflow`

创建新的工作流实例。

**请求体**:

```json
{
  "name": "我的视频项目",
  "input_type": "novel",
  "input_content": "小说内容或文件URL",
  "options": {
    "ai_provider": "gemini",
    "preferences": {}
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "workflow_id": "507f1f77bcf86cd799439011",
    "status": "pending",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 2.2 查询工作流

**GET** `/api/v1/workflow/:id`

查询工作流详情和当前状态。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "name": "我的视频项目",
    "status": "running",
    "current_stage": "storyboard",
    "progress": 0.5,
    "script": {
      "id": "script_001",
      "status": "completed",
      "screenplay": "剧本内容...",
      "storyboard_script": "分镜脚本..."
    },
    "assets": [],
    "storyboard": {
      "id": "storyboard_001",
      "status": "processing",
      "shots_count": 10,
      "completed_shots": 5
    },
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:30:00Z"
  }
}
```

### 2.3 查询工作流列表

**GET** `/api/v1/workflow`

查询用户的工作流列表。

**查询参数**:

- `page`: 页码 (默认: 1)
- `page_size`: 每页数量 (默认: 20)
- `status`: 状态筛选 (可选)
- `stage`: 阶段筛选 (可选)

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "workflows": [
      {
        "id": "507f1f77bcf86cd799439011",
        "name": "我的视频项目",
        "status": "running",
        "current_stage": "storyboard",
        "progress": 0.5,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 2.4 暂停工作流

**POST** `/api/v1/workflow/:id/pause`

暂停正在运行的工作流。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "status": "paused"
  }
}
```

### 2.5 恢复工作流

**POST** `/api/v1/workflow/:id/resume`

恢复暂停的工作流。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "status": "running"
  }
}
```

### 2.6 取消工作流

**POST** `/api/v1/workflow/:id/cancel`

取消工作流。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "status": "cancelled"
  }
}
```

### 2.7 查询工作流进度

**GET** `/api/v1/workflow/:id/progress`

查询工作流执行进度。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "workflow_id": "507f1f77bcf86cd799439011",
    "overall_progress": 0.5,
    "current_stage": "storyboard",
    "stage_progress": 0.6,
    "stages": [
      {
        "stage": "script",
        "status": "completed",
        "progress": 1.0
      },
      {
        "stage": "asset",
        "status": "completed",
        "progress": 1.0
      },
      {
        "stage": "storyboard",
        "status": "processing",
        "progress": 0.6
      }
    ]
  }
}
```

## 3. 剧本生成 API

### 3.1 创建剧本生成任务

**POST** `/api/v1/workflow/script`

创建剧本生成任务（通常由工作流自动调用，也可独立使用）。

**请求体**:

```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "input_type": "novel",
  "input_content": "小说内容...",
  "input_file_url": "https://storage.example.com/novel.txt",
  "options": {
    "ai_provider": "gemini",
    "model": "gemini-3-pro"
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_001",
    "status": "pending"
  }
}
```

### 3.2 查询剧本生成结果

**GET** `/api/v1/workflow/script/:id`

查询剧本生成结果。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "script_001",
    "input_type": "novel",
    "screenplay": "完整剧本内容...",
    "storyboard_script": "分镜头脚本...",
    "status": "completed",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 3.3 上传输入文件

**POST** `/api/v1/workflow/script/upload`

上传输入文件（小说、Word、PDF等）。

**请求**: `multipart/form-data`

- `file`: 文件
- `workflow_id`: 工作流ID

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "file_url": "https://storage.example.com/workflows/xxx/script/input/novel.txt",
    "file_size": 1024000,
    "content_type": "text/plain"
  }
}
```

## 4. 资产设计 API

### 4.1 创建资产设计任务

**POST** `/api/v1/workflow/assets`

创建资产设计任务。

**请求体**:

```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "script_id": "script_001",
  "options": {
    "ai_provider": "gemini",
    "image_provider": "seedream"
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_002",
    "status": "pending"
  }
}
```

### 4.2 查询资产列表

**GET** `/api/v1/workflow/assets`

查询工作流的资产列表。

**查询参数**:

- `workflow_id`: 工作流ID (必需)
- `type`: 资产类型筛选 (character/prop/scene)

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "assets": [
      {
        "id": "asset_001",
        "type": "character",
        "name": "主角",
        "description": "角色描述...",
        "prompt": "生成提示词...",
        "design_image_url": "https://storage.example.com/...",
        "status": "completed"
      }
    ],
    "total": 10
  }
}
```

### 4.3 上传参考图

**POST** `/api/v1/workflow/assets/:id/upload`

为资产上传参考图。

**请求**: `multipart/form-data`

- `file`: 图片文件
- `asset_id`: 资产ID

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "asset_id": "asset_001",
    "reference_image_url": "https://storage.example.com/...",
    "status": "pending"
  }
}
```

### 4.4 重新生成资产设计图

**POST** `/api/v1/workflow/assets/:id/regenerate`

重新生成资产的设计图。

**请求体**:

```json
{
  "prompt": "新的提示词（可选）",
  "options": {
    "ai_provider": "seedream"
  }
}
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "asset_id": "asset_001",
    "status": "processing"
  }
}
```

## 5. 分镜生成 API

### 5.1 创建分镜生成任务

**POST** `/api/v1/workflow/storyboard`

创建分镜生成任务。

**请求体**:

```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "script_id": "script_001",
  "options": {
    "grid_layout": "3x3",
    "ai_provider": "banana-pro",
    "auto_crop": true
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_003",
    "status": "pending"
  }
}
```

### 5.2 查询分镜详情

**GET** `/api/v1/workflow/storyboard/:id`

查询分镜详情。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "storyboard_001",
    "shots": [
      {
        "id": "shot_001",
        "sequence": 1,
        "prompt": "分镜提示词...",
        "shot_type": "close-up",
        "image_url": "https://storage.example.com/...",
        "cropped_image_url": "https://storage.example.com/...",
        "dialogue": "台词内容",
        "duration": 3.5
      }
    ],
    "grid_layout": "3x3",
    "grid_image_url": "https://storage.example.com/...",
    "status": "completed"
  }
}
```

### 5.3 编辑分镜提示词

**PUT** `/api/v1/workflow/storyboard/:id/shots/:shot_id`

编辑单个分镜的提示词。

**请求体**:

```json
{
  "prompt": "新的提示词",
  "replacements": {
    "character": "新角色描述",
    "scene": "新场景描述",
    "action": "新动作描述",
    "shot_type": "medium"
  }
}
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "shot_id": "shot_001",
    "prompt": "更新后的提示词",
    "status": "pending_regeneration"
  }
}
```

### 5.4 重新生成分镜

**POST** `/api/v1/workflow/storyboard/:id/regenerate`

重新生成分镜图。

**请求体**:

```json
{
  "shot_ids": ["shot_001", "shot_002"],
  "options": {
    "ai_provider": "banana-pro"
  }
}
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_004",
    "status": "pending"
  }
}
```

## 6. 动态分镜 API

### 6.1 创建动态分镜任务

**POST** `/api/v1/workflow/animatic`

创建动态分镜任务。

**请求体**:

```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "storyboard_id": "storyboard_001",
  "options": {
    "video_provider": "sora2",
    "audio_provider": "elevenlabs",
    "duration_control": {
      "min": 2.0,
      "max": 5.0
    }
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_005",
    "status": "pending"
  }
}
```

### 6.2 查询动态分镜详情

**GET** `/api/v1/workflow/animatic/:id`

查询动态分镜详情。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "animatic_001",
    "video_url": "https://storage.example.com/...",
    "audio_url": "https://storage.example.com/...",
    "shots": [
      {
        "shot_id": "shot_001",
        "video_url": "https://storage.example.com/...",
        "audio_url": "https://storage.example.com/...",
        "start_time": 0.0,
        "duration": 3.5
      }
    ],
    "total_duration": 120.0,
    "status": "completed"
  }
}
```

### 6.3 生成/上传音频

**POST** `/api/v1/workflow/animatic/:id/audio`

生成或上传音频。

**请求体** (生成音频):

```json
{
  "text": "台词内容",
  "shot_id": "shot_001",
  "options": {
    "provider": "elevenlabs",
    "voice_id": "voice_001"
  }
}
```

**请求** (上传音频): `multipart/form-data`

- `file`: 音频文件
- `shot_id`: 分镜ID

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "shot_id": "shot_001",
    "audio_url": "https://storage.example.com/...",
    "duration": 3.5
  }
}
```

## 7. 视频生成 API

### 7.1 创建视频生成任务

**POST** `/api/v1/workflow/video`

创建视频生成任务。

**请求体**:

```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "animatic_id": "animatic_001",
  "options": {
    "provider": "sora2",
    "quality": "high",
    "parallel": true
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_006",
    "status": "pending"
  }
}
```

### 7.2 查询视频生成进度

**GET** `/api/v1/workflow/video/:id/progress`

查询视频生成进度。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "video_id": "video_001",
    "progress": 0.6,
    "total_segments": 10,
    "completed_segments": 6,
    "segments": [
      {
        "id": "segment_001",
        "status": "completed",
        "video_url": "https://storage.example.com/..."
      },
      {
        "id": "segment_002",
        "status": "processing",
        "progress": 0.5
      }
    ]
  }
}
```

### 7.3 查询视频详情

**GET** `/api/v1/workflow/video/:id`

查询视频生成结果。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "video_001",
    "segments": [
      {
        "id": "segment_001",
        "animatic_shot_id": "shot_001",
        "video_url": "https://storage.example.com/...",
        "duration": 3.5,
        "status": "completed"
      }
    ],
    "status": "completed"
  }
}
```

## 8. 视频剪辑 API

### 8.1 创建剪辑任务

**POST** `/api/v1/workflow/edit`

创建视频剪辑任务。

**请求体**:

```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "video_id": "video_001",
  "options": {
    "add_audio_effects": true,
    "add_background_music": true,
    "music_style": "cinematic"
  }
}
```

**响应** (202 Accepted):

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_007",
    "status": "pending"
  }
}
```

### 8.2 添加背景音乐

**POST** `/api/v1/workflow/edit/:id/music`

为视频添加背景音乐。

**请求体**:

```json
{
  "music_url": "https://storage.example.com/music.mp3",
  "volume": 0.3,
  "fade_in": 2.0,
  "fade_out": 2.0
}
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "edit_id": "edit_001",
    "status": "processing"
  }
}
```

### 8.3 查询剪辑结果

**GET** `/api/v1/workflow/edit/:id`

查询视频剪辑结果。

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "edit_001",
    "final_video_url": "https://storage.example.com/final.mp4",
    "audio_effects": ["effect_001", "effect_002"],
    "background_music_url": "https://storage.example.com/music.mp3",
    "status": "completed",
    "duration": 120.0
  }
}
```

## 9. WebSocket 实时推送

### 9.1 连接工作流进度

**WebSocket** `/api/v1/workflow/:id/ws`

实时推送工作流进度更新。

**消息格式**:

```json
{
  "type": "progress",
  "data": {
    "workflow_id": "507f1f77bcf86cd799439011",
    "overall_progress": 0.5,
    "current_stage": "storyboard",
    "stage_progress": 0.6
  }
}
```

**消息类型**:

- `progress`: 进度更新
- `stage_completed`: 阶段完成
- `stage_failed`: 阶段失败
- `workflow_completed`: 工作流完成
- `workflow_failed`: 工作流失败

## 10. 错误码定义

| 错误码 | 说明 |
|--------|------|
| 40001 | 请求参数错误 |
| 40002 | 文件格式不支持 |
| 40003 | 文件大小超限 |
| 40101 | 未授权 |
| 40301 | 无权限访问该工作流 |
| 40401 | 工作流不存在 |
| 40402 | 资源不存在 |
| 50001 | 服务器内部错误 |
| 50002 | AI服务调用失败 |
| 50003 | 存储服务错误 |
| 50004 | 任务队列错误 |
