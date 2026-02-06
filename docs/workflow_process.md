# 工作流流程设计文档

## 流程概述

一个完整的工作流包含以下7个步骤，每一步用户都可以在页面上review和重新生成：

## 详细流程

### 第一步：上传剧本 + 基础设定（script）

**目标**：上传剧本文件，选择基础设定，解析和验证剧本

**操作**：
1. 用户上传剧本文件（TXT/DOC/DOCX/PDF）
2. 用户选择风格（漫剧/真人剧/混合）
3. 用户选择旁白类型（旁白/真人对话）
4. 系统解析剧本文件，提取元数据（书名、作者、简介）
5. 创建 Novel 记录和 Workflow 记录

**输出**：
- Novel 记录（包含书名、作者、简介、resource_id）
- Workflow 记录（包含风格、旁白类型等配置）

**当前实现**：
- ✅ `CreateWorkflow` - 创建工作流
- ✅ `CreateNovelFromResource` - 创建小说记录

---

### 第二步：切分章节，生成场景和分镜头，以及角色道具（storyboard）

**目标**：将剧本切分成章节，为每个章节生成场景、分镜头，并提取角色和道具

**操作**：
1. 根据 Novel 内容切分成多个章节（默认10章）
2. 为每个章节调用 LLM 生成结构化剧本（JSON格式），包含：
   - 场景（Scene）：场景编号、场景描述、图片提示词
   - 分镜头（Shot）：镜头编号、角色、画面描述、解说内容、图片提示词、视频提示词、运镜方式、时长
   - 角色（Character）：角色名称、性别、年龄组、角色编号
   - 道具（Prop）：道具名称、道具描述（如果有）
3. 保存 Chapter、Scene、Shot、Character、Prop 记录

**输出**：
- Chapter 记录（章节文本、统计信息）
- Scene 记录（场景信息）
- Shot 记录（分镜头信息，包含解说、图片提示词、视频提示词等）
- Character 记录（角色基本信息）
- Prop 记录（道具信息）

**当前实现**：
- ✅ `SplitNovelIntoChapters` - 切分章节
- ✅ `GenerateNarrationsForAllChapters` - 生成解说（包含场景和分镜头）
- ✅ `ConvertToScenesAndShots` - 转换场景、镜头、角色和道具
- ✅ 从 JSON 中提取完整的角色和道具信息（包括 description、image_prompt 等）

**已完成改进**：
- ✅ 从 LLM 返回的 JSON 中提取完整的角色信息（姓名、性别、年龄段、角色编号、描述、图片提示词）
- ✅ 从 LLM 返回的 JSON 中提取完整的道具信息（名称、描述、类别、图片提示词）
- ✅ 更新 LLM prompt，确保包含角色和道具的完整信息

---

### 第三步：生成图片（抽卡）（asset）

**目标**：根据角色、场景、道具生成相应的图片

**操作**：
1. 为每个角色生成角色图片（使用角色的图片提示词）
2. 为每个场景生成场景图片（使用场景的图片提示词）
3. 为每个道具生成道具图片（使用道具的图片提示词）
4. 为每个分镜头生成分镜头首图（使用分镜头的图片提示词）

**输出**：
- Character.ImageResourceID - 角色图片
- Scene.ImageResourceID - 场景图片
- Prop.ImageResourceID - 道具图片
- Shot.ImageResourceID - 分镜头首图

**当前实现**：
- ✅ `GenerateImagesForNarration` - 为解说生成分镜头图片
- ✅ `GenerateCharacterImages` - 生成角色图片
- ✅ `GenerateSceneImages` - 生成场景图片
- ✅ `GeneratePropImages` - 生成道具图片

**已完成改进**：
- ✅ 添加了角色、场景、道具图片生成功能
- ✅ 图片保存到 Character、Scene、Prop 的 ImageResourceID 字段
- ✅ 添加了对应的 API 接口（Handler 层）

---

### 第四步：完善分镜头脚本（storyboard refinement）

**目标**：完善分镜头的详细信息，包括解说、镜头描述、运镜手法、时长、图片提示词、视频提示词等

**操作**：
1. 用户可以在页面上review每个分镜头的脚本
2. 用户可以编辑：
   - 分镜头的解说（narration）
   - 镜头描述（image_prompt）
   - 运镜手法（camera_movement）
   - 时长（duration）
   - 镜头首图提示词（image_prompt）
   - 视频提示词（video_prompt）
3. 用户可以重新生成分镜头脚本（调用 LLM 优化）

**输出**：
- Shot 记录的更新（narration、image_prompt、video_prompt、camera_movement、duration）

**当前实现**：
- ✅ Shot 记录已包含这些字段
- ✅ `UpdateShot` API - 更新分镜头信息
- ✅ `RegenerateShotScript` API - 重新生成分镜头脚本（调用 LLM 优化）
- ✅ Handler 层接口已添加

**已完成改进**：
- ✅ 添加了更新分镜头的 API 接口（PUT `/api/v1/shots/:shot_id`）
- ✅ 添加了重新生成分镜头脚本的 API 接口（POST `/api/v1/shots/:shot_id/regenerate`）
- ⚠️ 前端页面展示和编辑功能待实现

---

### 第五步：生成音频（audio）

**目标**：根据分镜头的解说和时长生成音频

**操作**：
1. 为每个分镜头的解说内容生成 TTS 音频
2. 音频时长应该匹配分镜头的 duration（如果指定了）
3. 保存音频文件，记录音频时长

**输出**：
- Audio 记录（audio_resource_id、duration、sequence）

**当前实现**：
- ✅ `GenerateAudiosForNarration` - 为解说生成音频
- ✅ `generateSingleAudio` - 生成单个音频

**需要改进**：
- 确保音频时长与分镜头的 duration 匹配
- 用户可以在页面上review每个音频，选择重新生成

---

### 第六步：生成分镜视频（video）

**目标**：根据音频、分镜头脚本图片、角色图片、场景图片、道具图片等生成对应时长的分镜视频

**操作**：
1. 为每个分镜头生成视频：
   - 使用分镜头的首图（Shot.ImageResourceID）
   - 如果分镜头需要角色图片，使用角色的图片（Character.ImageResourceID）
   - 如果分镜头需要场景图片（远景），使用场景的图片（Scene.ImageResourceID）
   - 如果分镜头需要道具图片，使用道具的图片（Prop.ImageResourceID）
   - 使用分镜头的视频提示词（video_prompt）
   - 使用分镜头的音频（Audio）
   - 视频时长应该匹配音频时长
2. 保存视频文件

**输出**：
- Video 记录（video_resource_id、duration、sequence）

**当前实现**：
- ✅ `GenerateNarrationVideosForChapter` - 为章节生成解说视频
- ✅ `generateSingleNarrationVideo` - 生成单个分镜视频
- ✅ 视频生成使用分镜头图片（Image 表）

**已完成改进**：
- ✅ 简化了视频生成逻辑，直接使用分镜头图片
- ⚠️ 后续可以根据需要扩展，支持使用角色、场景、道具图片（当前逻辑已简化）
- ⚠️ 前端页面 review 和重新生成功能待实现

---

### 第七步：合并成完整视频（edit）

**目标**：将所有分镜视频合并成完整视频

**操作**：
1. 获取章节的所有分镜视频（按 sequence 排序）
2. 使用 FFmpeg 合并所有视频
3. 添加字幕（如果有）
4. 保存最终视频

**输出**：
- FinalVideo 记录（final_video_resource_id）

**当前实现**：
- ✅ `GenerateFinalVideoForChapter` - 生成最终视频
- ✅ `generateMergedNarrationVideo` - 合并视频

**需要改进**：
- 用户可以在页面上review最终视频，选择重新合并

---

## 工作流阶段映射

| 用户期望的步骤 | 工作流阶段（WorkflowStage） | 说明 |
|--------------|---------------------------|------|
| 第一步：上传剧本 + 基础设定 | `script` | 创建 Novel 和 Workflow |
| 第二步：切分章节，生成场景和分镜头 | `storyboard` | 生成 Chapter、Scene、Shot、Character、Prop |
| 第三步：生成图片（抽卡） | `asset` | 生成角色、场景、道具、分镜头图片 |
| 第四步：完善分镜头脚本 | `storyboard` | 用户review和编辑分镜头脚本 |
| 第五步：生成音频 | `asset` | 生成 TTS 音频 |
| 第六步：生成分镜视频 | `video` | 生成分镜视频 |
| 第七步：合并成完整视频 | `edit` | 合并所有分镜视频 |

## 当前实现状态

### ✅ 已实现（后端）
1. ✅ 第一步：上传剧本 + 基础设定
2. ✅ 第二步：切分章节，生成场景和分镜头，提取角色和道具
3. ✅ 第三步：生成图片（角色、场景、道具、分镜头）
4. ✅ 第四步：更新分镜头脚本、重新生成分镜头脚本
5. ✅ 第五步：生成音频
6. ✅ 第六步：生成分镜视频
7. ✅ 第七步：合并成完整视频

### ✅ 已实现的 API 接口
1. ✅ `PUT /api/v1/shots/:shot_id` - 更新分镜头信息
2. ✅ `POST /api/v1/shots/:shot_id/regenerate` - 重新生成分镜头脚本
3. ✅ `POST /api/v1/novels/:novel_id/characters/images` - 生成角色图片
4. ✅ `POST /api/v1/narrations/:narration_id/scenes/images` - 生成场景图片
5. ✅ `POST /api/v1/novels/:novel_id/props/images` - 生成道具图片

### ⚠️ 待实现（前端）
1. ⚠️ 前端页面：每一步的 review 和重新生成功能
2. ⚠️ 前端页面：分镜头脚本编辑界面
3. ⚠️ 前端页面：图片 review 和重新生成界面
4. ⚠️ 前端页面：音频 review 和重新生成界面
5. ⚠️ 前端页面：视频 review 和重新生成界面

## 下一步计划

1. **改进第二步**：从 LLM 返回的 JSON 中提取角色和道具信息
2. **实现第三步**：添加角色、场景、道具图片生成功能
3. **实现第四步**：添加分镜头脚本编辑和重新生成功能
4. **改进第六步**：视频生成时使用角色、场景、道具图片
5. **实现前端页面**：每一步的 review 和重新生成功能

