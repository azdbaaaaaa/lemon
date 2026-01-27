# Narration JSON格式规范文档

## 1. 概述

本文档定义了Narration模块使用的JSON格式规范，用于存储和传输解说文案数据。该格式保持了章节->分镜的层次结构，便于处理和扩展。

## 2. JSON Schema定义

### 2.1 完整Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Narration Script",
  "description": "解说文案JSON格式定义",
  "type": "object",
  "required": ["version", "workflow_id", "chapters"],
  "properties": {
    "version": {
      "type": "string",
      "description": "JSON格式版本号",
      "pattern": "^\\d+\\.\\d+$",
      "example": "1.0"
    },
    "workflow_id": {
      "type": "string",
      "description": "所属工作流ID",
      "pattern": "^[a-zA-Z0-9_-]+$"
    },
    "resource_id": {
      "type": "string",
      "description": "关联的资源ID（指向输入文件资源）",
      "pattern": "^[a-zA-Z0-9_-]+$"
    },
    "metadata": {
      "type": "object",
      "description": "元数据对象",
      "properties": {
        "title": {
          "type": "string",
          "description": "标题"
        },
        "author": {
          "type": "string",
          "description": "作者名"
        },
        "generated_at": {
          "type": "string",
          "format": "date-time",
          "description": "生成时间（ISO 8601格式）"
        },
        "generated_by": {
          "type": "string",
          "description": "生成服务名称",
          "default": "narration_service"
        },
        "ai_provider": {
          "type": "string",
          "description": "AI服务提供商",
          "enum": ["gemini", "doubao", "openai"]
        },
        "ai_model": {
          "type": "string",
          "description": "AI模型名称",
          "example": "gemini-3-pro"
        },
        "total_chapters": {
          "type": "integer",
          "description": "总章节数",
          "minimum": 0
        },
        "total_shots": {
          "type": "integer",
          "description": "总分镜数",
          "minimum": 0
        },
        "status": {
          "type": "string",
          "description": "状态",
          "enum": ["pending", "generating", "completed", "validating", "validated", "failed"]
        }
      }
    },
    "chapters": {
      "type": "array",
      "description": "章节数组",
      "minItems": 0,
      "items": {
        "$ref": "#/definitions/Chapter"
      }
    },
    "validation_report": {
      "type": "object",
      "description": "验证报告（可选，在验证后添加）",
      "properties": {
        "validated_at": {
          "type": "string",
          "format": "date-time",
          "description": "验证时间"
        },
        "total_shots": {
          "type": "integer",
          "description": "总分镜数"
        },
        "valid_shots": {
          "type": "integer",
          "description": "符合要求的分镜数"
        },
        "invalid_shots": {
          "type": "integer",
          "description": "不符合要求的分镜数"
        },
        "fixed_shots": {
          "type": "integer",
          "description": "已修复的分镜数"
        },
        "failed_shots": {
          "type": "integer",
          "description": "修复失败的分镜数"
        },
        "statistics": {
          "type": "object",
          "description": "统计信息",
          "properties": {
            "min_words": {
              "type": "integer",
              "description": "最小字数"
            },
            "max_words": {
              "type": "integer",
              "description": "最大字数"
            },
            "average_words": {
              "type": "number",
              "description": "平均字数"
            },
            "median_words": {
              "type": "number",
              "description": "中位数字数"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Chapter": {
      "type": "object",
      "description": "章节对象",
      "required": ["id", "sequence", "title", "shots"],
      "properties": {
        "id": {
          "type": "string",
          "description": "章节唯一标识",
          "pattern": "^chapter_[a-zA-Z0-9_-]+$"
        },
        "sequence": {
          "type": "integer",
          "description": "章节序号（从1开始）",
          "minimum": 1
        },
        "title": {
          "type": "string",
          "description": "章节标题",
          "minLength": 1
        },
        "metadata": {
          "type": "object",
          "description": "章节元数据",
          "properties": {
            "original_title": {
              "type": "string",
              "description": "原始章节标题"
            },
            "word_count": {
              "type": "integer",
              "description": "章节总字数",
              "minimum": 0
            },
            "shot_count": {
              "type": "integer",
              "description": "章节分镜数",
              "minimum": 0
            }
          }
        },
        "shots": {
          "type": "array",
          "description": "分镜数组",
          "minItems": 0,
          "items": {
            "$ref": "#/definitions/Shot"
          }
        }
      }
    },
    "Shot": {
      "type": "object",
      "description": "分镜对象",
      "required": ["id", "sequence", "scene", "narration", "shot_type"],
      "properties": {
        "id": {
          "type": "string",
          "description": "分镜唯一标识",
          "pattern": "^shot_[a-zA-Z0-9_-]+$"
        },
        "sequence": {
          "type": "integer",
          "description": "分镜序号（在章节内从1开始）",
          "minimum": 1
        },
        "scene": {
          "type": "string",
          "description": "场景描述",
          "minLength": 1
        },
        "narration": {
          "type": "string",
          "description": "解说文案（必需）",
          "minLength": 1
        },
        "dialogue": {
          "type": ["string", "null"],
          "description": "台词（可选，null表示无台词）"
        },
        "shot_type": {
          "type": "string",
          "description": "景别类型",
          "enum": ["close-up", "medium", "wide", "extreme-close-up", "extreme-wide"]
        },
        "metadata": {
          "type": "object",
          "description": "分镜元数据",
          "properties": {
            "word_count": {
              "type": "integer",
              "description": "字数统计",
              "minimum": 0
            },
            "duration_estimate": {
              "type": "number",
              "description": "预估时长（秒）",
              "minimum": 0
            },
            "status": {
              "type": "string",
              "description": "验证状态",
              "enum": ["valid", "too_short", "too_long", "invalid"]
            },
            "validated_at": {
              "type": "string",
              "format": "date-time",
              "description": "验证时间"
            },
            "fixed": {
              "type": "boolean",
              "description": "是否已修复"
            },
            "fix_type": {
              "type": "string",
              "description": "修复类型",
              "enum": ["shorten", "expand"]
            }
          }
        }
      }
    }
  }
}
```

## 3. 完整示例

### 3.1 基础示例

```json
{
  "version": "1.0",
  "workflow_id": "workflow_001",
  "resource_id": "resource_001",
  "metadata": {
    "title": "小说标题",
    "author": "作者名",
    "generated_at": "2024-01-01T10:00:00Z",
    "generated_by": "narration_service",
    "ai_provider": "gemini",
    "ai_model": "gemini-3-pro",
    "total_chapters": 10,
    "total_shots": 150,
    "status": "completed"
  },
  "chapters": [
    {
      "id": "chapter_1",
      "sequence": 1,
      "title": "第一章",
      "metadata": {
        "original_title": "第一章 开始",
        "word_count": 5000,
        "shot_count": 15
      },
      "shots": [
        {
          "id": "shot_1",
          "sequence": 1,
          "scene": "场景描述：一个安静的图书馆，阳光透过窗户洒在书桌上",
          "narration": "在这个安静的图书馆里，时间仿佛静止了。阳光透过窗户，在书桌上投下斑驳的光影。",
          "dialogue": null,
          "shot_type": "wide",
          "metadata": {
            "word_count": 45,
            "duration_estimate": 3.5,
            "status": "valid",
            "validated_at": "2024-01-01T10:05:00Z"
          }
        },
        {
          "id": "shot_2",
          "sequence": 2,
          "scene": "场景描述：特写镜头，聚焦在一本翻开的书上",
          "narration": "书页上密密麻麻的文字，记录着古老的故事。",
          "dialogue": null,
          "shot_type": "close-up",
          "metadata": {
            "word_count": 18,
            "duration_estimate": 2.0,
            "status": "too_short",
            "validated_at": "2024-01-01T10:05:00Z",
            "fixed": true,
            "fix_type": "expand"
          }
        }
      ]
    }
  ],
  "validation_report": {
    "validated_at": "2024-01-01T10:10:00Z",
    "total_shots": 150,
    "valid_shots": 145,
    "invalid_shots": 5,
    "fixed_shots": 5,
    "failed_shots": 0,
    "statistics": {
      "min_words": 20,
      "max_words": 100,
      "average_words": 52.3,
      "median_words": 48
    }
  }
}
```

### 3.2 带台词的示例

```json
{
  "version": "1.0",
  "workflow_id": "workflow_002",
  "resource_id": "resource_002",
  "metadata": {
    "title": "对话场景",
    "generated_at": "2024-01-01T11:00:00Z",
    "generated_by": "narration_service",
    "ai_provider": "gemini",
    "total_chapters": 1,
    "total_shots": 3,
    "status": "completed"
  },
  "chapters": [
    {
      "id": "chapter_1",
      "sequence": 1,
      "title": "对话场景",
      "metadata": {
        "word_count": 500,
        "shot_count": 3
      },
      "shots": [
        {
          "id": "shot_1",
          "sequence": 1,
          "scene": "场景描述：两个人在咖啡厅里对话",
          "narration": "午后的咖啡厅，阳光透过落地窗洒在桌面上。",
          "dialogue": "你好，好久不见。",
          "shot_type": "medium",
          "metadata": {
            "word_count": 25,
            "duration_estimate": 4.0,
            "status": "valid"
          }
        },
        {
          "id": "shot_2",
          "sequence": 2,
          "scene": "场景描述：特写镜头，聚焦在说话人的脸上",
          "narration": "他的脸上露出了温暖的笑容。",
          "dialogue": "是啊，确实很久了。",
          "shot_type": "close-up",
          "metadata": {
            "word_count": 20,
            "duration_estimate": 3.5,
            "status": "valid"
          }
        }
      ]
    }
  ]
}
```

## 4. 字段说明

### 4.1 根级别字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `version` | string | 是 | JSON格式版本号，格式：`major.minor` |
| `workflow_id` | string | 是 | 所属工作流ID |
| `resource_id` | string | 否 | 关联的资源ID（指向输入文件资源） |
| `metadata` | object | 否 | 元数据对象 |
| `chapters` | array | 是 | 章节数组，至少包含一个章节 |
| `validation_report` | object | 否 | 验证报告（在验证后添加） |

### 4.2 Metadata字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `title` | string | 否 | 标题 |
| `author` | string | 否 | 作者名 |
| `generated_at` | string (ISO 8601) | 否 | 生成时间 |
| `generated_by` | string | 否 | 生成服务名称，默认：`narration_service` |
| `ai_provider` | string | 否 | AI服务提供商（gemini/doubao/openai） |
| `ai_model` | string | 否 | AI模型名称 |
| `total_chapters` | integer | 否 | 总章节数 |
| `total_shots` | integer | 否 | 总分镜数 |
| `status` | string | 否 | 状态（pending/generating/completed/validating/validated/failed） |

### 4.3 Chapter字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 章节唯一标识，格式：`chapter_{identifier}` |
| `sequence` | integer | 是 | 章节序号，从1开始，必须连续 |
| `title` | string | 是 | 章节标题 |
| `metadata` | object | 否 | 章节元数据 |
| `shots` | array | 是 | 分镜数组，至少包含一个分镜 |

### 4.4 Chapter Metadata字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `original_title` | string | 否 | 原始章节标题 |
| `word_count` | integer | 否 | 章节总字数 |
| `shot_count` | integer | 否 | 章节分镜数 |

### 4.5 Shot字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 分镜唯一标识，格式：`shot_{identifier}` |
| `sequence` | integer | 是 | 分镜序号，在章节内从1开始，必须连续 |
| `scene` | string | 是 | 场景描述 |
| `narration` | string | 是 | 解说文案，字数应在20-100字之间（验证后） |
| `dialogue` | string\|null | 否 | 台词，null表示无台词 |
| `shot_type` | string | 是 | 景别类型（close-up/medium/wide/extreme-close-up/extreme-wide） |
| `metadata` | object | 否 | 分镜元数据 |

### 4.6 Shot Metadata字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `word_count` | integer | 否 | 字数统计 |
| `duration_estimate` | number | 否 | 预估时长（秒） |
| `status` | string | 否 | 验证状态（valid/too_short/too_long/invalid） |
| `validated_at` | string (ISO 8601) | 否 | 验证时间 |
| `fixed` | boolean | 否 | 是否已修复 |
| `fix_type` | string | 否 | 修复类型（shorten/expand） |

### 4.7 Validation Report字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `validated_at` | string (ISO 8601) | 是 | 验证时间 |
| `total_shots` | integer | 是 | 总分镜数 |
| `valid_shots` | integer | 是 | 符合要求的分镜数 |
| `invalid_shots` | integer | 是 | 不符合要求的分镜数 |
| `fixed_shots` | integer | 是 | 已修复的分镜数 |
| `failed_shots` | integer | 是 | 修复失败的分镜数 |
| `statistics` | object | 否 | 统计信息 |

## 5. 数据验证规则

### 5.1 必需字段验证

- 根级别：`version`, `workflow_id`, `chapters` 为必需字段
- Chapter级别：`id`, `sequence`, `title`, `shots` 为必需字段
- Shot级别：`id`, `sequence`, `scene`, `narration`, `shot_type` 为必需字段

### 5.2 顺序验证

- `chapters` 数组中的章节必须按 `sequence` 字段排序（1, 2, 3...）
- 每个章节的 `shots` 数组必须按 `sequence` 字段排序（1, 2, 3...）
- 序号必须连续，不能有缺失

### 5.3 ID唯一性验证

- 所有章节ID必须在文档范围内唯一
- 所有分镜ID必须在文档范围内唯一
- ID格式必须符合正则表达式：`^chapter_[a-zA-Z0-9_-]+$` 和 `^shot_[a-zA-Z0-9_-]+$`

### 5.4 字数验证

- `narration` 字段的字数应在20-100字之间（验证后）
- 字数统计规则：
  - 中文字符：每个字符计1字
  - 英文单词：每个单词计1字（空格分隔）
  - 标点符号：不计入字数
  - 数字：每个数字计1字

### 5.5 格式验证

- `version` 必须符合格式：`major.minor`（如：`1.0`, `1.1`）
- `generated_at` 和 `validated_at` 必须符合ISO 8601格式
- `shot_type` 必须是枚举值之一

## 6. 版本管理

### 6.1 版本号规则

- 格式：`major.minor`
- `major`：主版本号，不兼容的格式变更时递增
- `minor`：次版本号，向后兼容的功能添加时递增

### 6.2 版本历史

- **1.0**（当前版本）：
  - 初始版本
  - 支持章节和分镜的层次结构
  - 支持验证报告

## 7. 扩展性考虑

### 7.1 可选字段

- 所有 `metadata` 字段都是可选的，便于扩展
- `dialogue` 字段为可选，支持无台词的场景
- `validation_report` 为可选，仅在验证后添加

### 7.2 自定义字段

- 可以在 `metadata` 对象中添加自定义字段
- 建议使用命名空间前缀避免冲突（如：`custom_*`）

### 7.3 向后兼容

- 新版本应保持向后兼容
- 新增字段应为可选字段
- 废弃字段应保留但标记为deprecated

## 8. 使用建议

### 8.1 生成阶段

- 生成时可以不包含 `validation_report`
- `metadata.status` 应设置为 `"generating"` 或 `"completed"`
- 所有必需字段必须填充

### 8.2 验证阶段

- 验证后应添加 `validation_report`
- 更新 `metadata.status` 为 `"validated"`
- 更新每个分镜的 `metadata.status` 和 `metadata.validated_at`

### 8.3 修复阶段

- 修复后应更新修复的分镜的 `metadata.fixed` 和 `metadata.fix_type`
- 更新 `validation_report` 中的修复统计

## 9. 相关文档

- [Narration模块设计文档](./NARRATION_MODULE_DESIGN.md) - 完整的模块设计
- [资源模块设计文档](./RESOURCE_MODULE_DESIGN.md) - 资源管理模块设计
