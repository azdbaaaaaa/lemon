# 资源模块设计文档 (DSPEC)

## 1. 模块概述

### 1.1 功能定位

资源模块是一个**独立的基础设施模块**，负责所有资源的存储、下载、元数据管理和访问控制。该模块**不依赖workflow**，为其他业务模块（如Narration、Storyboard、Video、Workflow等）提供统一的资源管理服务。

**重要设计原则**：
- ✅ 资源模块是独立的，只负责资源的存储和管理
- ✅ 资源不直接关联workflow，通过resource_id在业务模块中建立关联
- ✅ 如果workflow需要使用资源，应在workflow模块中维护resource_id列表
- ✅ 资源模块不区分分类（input/intermediate/output等），这些由具体的业务模块来决定
- ✅ 资源模块只记录文件扩展名（如：txt、pdf、jpg、mp4等），不维护预定义的类型枚举
- ✅ 资源模块不包含文件处理逻辑，处理由业务模块（如narration）完成

### 1.2 核心特性

- **统一资源管理**: 提供统一的资源存储和管理接口
- **多存储后端支持**: 支持AWS S3、阿里云OSS、MinIO、本地文件系统等多种存储后端
- **客户端直传**: 采用客户端上传方案，减少服务器负载
- **元数据管理**: 完整的资源元数据管理和查询能力
- **访问控制**: 基于用户权限的访问控制
- **版本管理**: 支持资源版本管理和备份
- **可插拔架构**: 根据配置自动选择存储实现，易于扩展

### 1.3 与其他模块的关系

```
业务模块（Narration/Storyboard/Video等）
    ↓ (请求资源)
资源模块（Resource Module）
    ↓ (存储/下载)
存储服务（S3/OSS/MinIO/Local）
```

## 2. 数据库结构设计

### 2.1 Resource 实体

```go
// internal/model/resource/resource.go
package resource

import (
    "time"
)

// Resource 资源实体
// 注意: 资源模块是独立的，不依赖workflow。如果需要在workflow中使用资源，应在workflow模块中维护resource_id关联关系。
// 资源模块不区分分类（input/intermediate/output等），这些由具体的业务模块来决定。
// 资源类型只记录文件扩展名（如：txt、pdf、jpg、mp4等），不维护预定义的类型枚举。
type Resource struct {
    ID          string                 `bson:"id" json:"id"`                       // 资源ID（UUID）
    UserID      string                 `bson:"user_id" json:"user_id"`           // 所属用户ID
    Ext         string                 `bson:"ext" json:"ext"`                    // 文件扩展名（不含点号，如：txt、pdf、jpg、mp4等）
    Name        string                 `bson:"name" json:"name"`                  // 原始文件名
    DisplayName string                 `bson:"display_name,omitempty" json:"display_name,omitempty"` // 显示名称
    Description string                 `bson:"description,omitempty" json:"description,omitempty"` // 描述
    
    // 存储信息
    StorageKey  string                 `bson:"storage_key" json:"storage_key"`     // 存储路径（key）
    StorageURL  string                 `bson:"storage_url,omitempty" json:"storage_url,omitempty"` // 存储URL（临时访问）
    StorageType string                 `bson:"storage_type" json:"storage_type"`   // 存储类型（s3/oss/minio/local）
    
    // 文件信息
    FileSize    int64                  `bson:"file_size" json:"file_size"`        // 文件大小（字节）
    ContentType string                 `bson:"content_type" json:"content_type"`  // MIME类型
    MD5         string                 `bson:"md5,omitempty" json:"md5,omitempty"` // 文件MD5值（用于去重）
    SHA256      string                 `bson:"sha256,omitempty" json:"sha256,omitempty"` // 文件SHA256值
    
    // 元数据
    Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"` // 扩展元数据
    Tags        []string               `bson:"tags,omitempty" json:"tags,omitempty"` // 标签
    
    // 版本管理
    Version     int                    `bson:"version" json:"version"`            // 版本号
    ParentID    string                 `bson:"parent_id,omitempty" json:"parent_id,omitempty"` // 父资源ID（用于版本链）
    
    // 状态
    Status      ResourceStatus         `bson:"status" json:"status"`              // 资源状态
    
    // 时间戳
    UploadedAt  time.Time              `bson:"uploaded_at" json:"uploaded_at"`    // 上传时间
    CreatedAt   time.Time              `bson:"created_at" json:"created_at"`      // 创建时间
    UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`      // 更新时间
    DeletedAt   *time.Time             `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"` // 软删除时间
}

// ResourceStatus 资源状态
// 注意: 资源模块不区分分类（input/intermediate/output等），这些由具体的业务模块来决定。
// 资源类型只记录文件扩展名（如：txt、pdf、jpg、mp4等），不维护预定义的类型枚举。
type ResourceStatus string

const (
    ResourceStatusUploading   ResourceStatus = "uploading"   // 上传中
    ResourceStatusUploaded    ResourceStatus = "uploaded"    // 已上传
    ResourceStatusReady       ResourceStatus = "ready"       // 就绪
    ResourceStatusFailed      ResourceStatus = "failed"      // 失败
    ResourceStatusDeleted     ResourceStatus = "deleted"    // 已删除
)
```

### 2.2 UploadSession 实体（客户端上传会话）

```go
// UploadSession 上传会话（用于客户端直传）
type UploadSession struct {
    ID          string                 `bson:"id" json:"id"`                       // 会话ID（UUID）
    UserID      string                 `bson:"user_id" json:"user_id"`           // 所属用户ID
    
    // 文件信息
    FileName    string                 `bson:"file_name" json:"file_name"`
    FileSize    int64                  `bson:"file_size" json:"file_size"`
    ContentType string                 `bson:"content_type" json:"content_type"`
    Ext         string                 `bson:"ext" json:"ext"`                    // 文件扩展名（不含点号，如：txt、pdf、jpg、mp4等）
    
    // 客户端上传相关
    UploadURL   string                 `bson:"upload_url" json:"upload_url"`     // 客户端上传URL（预签名URL）
    UploadKey   string                 `bson:"upload_key" json:"upload_key"`     // 上传路径（key）
    ExpiresAt   time.Time              `bson:"expires_at" json:"expires_at"`      // 上传URL过期时间
    
    // 上传状态
    Status      UploadStatus           `bson:"status" json:"status"`              // 上传状态
    UploadedBytes int64                `bson:"uploaded_bytes" json:"uploaded_bytes"` // 已上传字节数
    ResourceID  string                 `bson:"resource_id,omitempty" json:"resource_id,omitempty"` // 上传完成后的资源ID
    
    // 时间戳
    CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

// UploadStatus 上传状态
type UploadStatus string

const (
    UploadStatusPending   UploadStatus = "pending"   // 待上传
    UploadStatusUploading UploadStatus = "uploading" // 上传中
    UploadStatusCompleted UploadStatus = "completed" // 上传完成
    UploadStatusFailed    UploadStatus = "failed"    // 上传失败
    UploadStatusExpired   UploadStatus = "expired"   // 已过期
)
```

### 2.3 数据库索引

```go
// MongoDB索引定义
// 注意: 资源模块是独立的，不依赖workflow，也不区分分类（category），这些由业务模块决定

// 1. 用户ID + 状态（常用查询）
db.resources.createIndex({ user_id: 1, status: 1 })

// 2. 用户ID + 创建时间（用户资源列表）
db.resources.createIndex({ user_id: 1, created_at: -1 })

// 3. MD5值（去重查询）
db.resources.createIndex({ md5: 1 })

// 4. 存储路径（快速查找）
db.resources.createIndex({ storage_key: 1 })

// 5. 文件扩展名（扩展名查询）
db.resources.createIndex({ ext: 1 })

// 6. 上传会话 - 用户ID + 状态
db.upload_sessions.createIndex({ user_id: 1, status: 1 })

// 7. 上传会话 - 过期时间（清理过期会话）
db.upload_sessions.createIndex({ expires_at: 1 }, { expireAfterSeconds: 0 })
```

## 3. API接口设计

### 3.1 获取上传凭证（客户端直传）

**POST** `/api/v1/resources/upload/prepare`

创建上传会话，返回客户端直传URL。

**请求体**:
```json
{
  "file_name": "novel.txt",
  "file_size": 1024000,
  "content_type": "text/plain",
  "ext": "txt"
}
```

**注意**: 资源模块是独立的，不依赖workflow。如果需要在workflow中使用资源，应在workflow模块中维护resource_id关联关系。

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "session_001",
    "upload_url": "https://storage.example.com/upload?signature=...",
    "upload_key": "resources/user_001/input/novel.txt",
    "expires_at": "2024-01-01T12:00:00Z",
    "upload_method": "PUT",
    "headers": {
      "Content-Type": "text/plain",
      "Content-Length": "1024000"
    }
  }
}
```

### 3.2 确认上传完成

**POST** `/api/v1/resources/upload/complete`

客户端上传完成后，通知服务器确认。

**请求体**:
```json
{
  "session_id": "session_001",
  "md5": "d41d8cd98f00b204e9800998ecf8427e",
  "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "resource_id": "resource_001",
    "resource_url": "https://storage.example.com/resources/user_001/input/novel.txt",
    "file_size": 1024000
  }
}
```

### 3.3 查询资源列表

**GET** `/api/v1/resources`

查询资源列表。

**查询参数**:
- `user_id`: 用户ID（可选，管理员可查询所有用户资源）
- `ext`: 文件扩展名筛选（如：txt、pdf、jpg、mp4等）
- `status`: 状态筛选
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认20）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "resources": [
      {
        "id": "resource_001",
        "user_id": "user_001",
        "ext": "txt",
        "name": "novel.txt",
        "display_name": "小说文档",
        "file_size": 1024000,
        "content_type": "text/plain",
        "status": "ready",
        "uploaded_at": "2024-01-01T10:00:00Z",
        "created_at": "2024-01-01T10:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 3.4 获取下载URL

**GET** `/api/v1/resources/download`

获取资源下载URL（预签名URL）。

**查询参数**:
- `resource_id`: 资源ID（必需）
- `expires_in`: URL有效期（秒，默认3600）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "resource_id": "resource_001",
    "download_url": "https://storage.example.com/resources/user_001/input/novel.txt?signature=...",
    "expires_at": "2024-01-01T13:00:00Z",
    "file_name": "novel.txt",
    "file_size": 1024000,
    "content_type": "text/plain"
  }
}
```

### 3.5 删除资源

**POST** `/api/v1/resources/delete`

删除资源（软删除）。

**请求体**:
```json
{
  "resource_id": "resource_001"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "删除成功"
}
```

### 3.6 查询资源详情

**GET** `/api/v1/resources`

查询资源详情。

**查询参数**:
- `resource_id`: 资源ID（必需）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data":           {
            "id": "resource_001",
            "user_id": "user_001",
    "ext": "txt",
    "name": "novel.txt",
    "display_name": "小说文档",
    "description": "用户上传的小说文档",
    "storage_key": "resources/user_001/input/novel.txt",
    "storage_type": "s3",
    "file_size": 1024000,
    "content_type": "text/plain",
    "md5": "d41d8cd98f00b204e9800998ecf8427e",
    "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "metadata": {
      "author": "作者名",
      "title": "小说标题"
    },
    "tags": ["novel", "fiction"],
    "version": 1,
    "status": "ready",
    "uploaded_at": "2024-01-01T10:00:00Z",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z"
  }
}
```

## 4. 分层架构设计

### 4.1 Handler层

**文件**: `internal/handler/resource/`

**职责**:
- HTTP请求解析和参数校验
- 响应封装和错误处理
- 请求路由

**主要文件**:
- `handler.go`: Handler结构体定义
- `upload.go`: 上传相关接口
- `download.go`: 下载相关接口
- `list.go`: 列表查询接口
- `detail.go`: 详情查询接口
- `delete.go`: 删除接口

### 4.2 Service层

**文件**: `internal/service/resource_service.go`

**职责**:
- 业务流程编排
- 业务规则校验
- 调用Repository和Storage层

**主要方法**:
```go
// PrepareUpload 准备上传（创建上传会话）
func (s *ResourceService) PrepareUpload(ctx context.Context, req *PrepareUploadRequest) (*UploadSession, error)

// CompleteUpload 完成上传（确认上传完成）
func (s *ResourceService) CompleteUpload(ctx context.Context, sessionID string, md5, sha256 string) (*Resource, error)

// GetDownloadURL 获取下载URL
func (s *ResourceService) GetDownloadURL(ctx context.Context, resourceID string, expiresIn time.Duration) (string, error)

// ListResources 查询资源列表
func (s *ResourceService) ListResources(ctx context.Context, req *ListResourcesRequest) (*ListResourcesResult, error)

// GetResource 获取资源详情
func (s *ResourceService) GetResource(ctx context.Context, resourceID string) (*Resource, error)

// DeleteResource 删除资源
func (s *ResourceService) DeleteResource(ctx context.Context, resourceID string) error
```

### 4.3 Repository层

**文件**: `internal/repository/resource_repo.go`

**职责**:
- 数据库操作封装
- CRUD操作
- 查询优化

**主要方法**:
```go
// Create 创建资源
func (r *ResourceRepo) Create(ctx context.Context, resource *Resource) error

// FindByID 根据ID查询
func (r *ResourceRepo) FindByID(ctx context.Context, id string) (*Resource, error)

// FindByUserID 根据用户ID查询资源列表
func (r *ResourceRepo) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Resource, int64, error)

// FindByMD5 根据MD5查询（去重）
func (r *ResourceRepo) FindByMD5(ctx context.Context, md5 string) (*Resource, error)

// Update 更新资源
func (r *ResourceRepo) Update(ctx context.Context, id string, updates map[string]interface{}) error

// Delete 删除资源（软删除）
func (r *ResourceRepo) Delete(ctx context.Context, id string) error

// CreateUploadSession 创建上传会话
func (r *ResourceRepo) CreateUploadSession(ctx context.Context, session *UploadSession) error

// FindUploadSession 查询上传会话
func (r *ResourceRepo) FindUploadSession(ctx context.Context, sessionID string) (*UploadSession, error)

// UpdateUploadSession 更新上传会话
func (r *ResourceRepo) UpdateUploadSession(ctx context.Context, sessionID string, updates map[string]interface{}) error
```

### 4.4 Storage层（存储抽象层）

**文件**: `internal/pkg/storage/`

**职责**:
- 文件存储抽象
- 支持多种对象存储服务（S3、OSS、MinIO等）和本地文件系统
- 根据配置自动选择存储实现
- 预签名URL生成

**接口定义**:
```go
// Storage 存储接口
type Storage interface {
    // 上传文件（服务端上传）
    Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
    
    // 下载文件
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    
    // 获取预签名上传URL（客户端直传）
    GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error)
    
    // 获取预签名下载URL
    GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
    
    // 删除文件
    Delete(ctx context.Context, key string) error
    
    // 检查文件是否存在
    Exists(ctx context.Context, key string) (bool, error)
    
    // 获取文件信息
    GetFileInfo(ctx context.Context, key string) (*FileInfo, error)
    
    // 获取存储类型
    GetStorageType() string
}

type FileInfo struct {
    Key         string
    Size        int64
    ContentType string
    ETag        string
    LastModified time.Time
}
```

## 5. 存储路径规范

资源模块是独立的，存储路径按用户组织，不依赖workflow，也不区分分类：

```
resources/
├── {user_id}/             # 用户资源目录
│   ├── {resource_id}.{ext}  # 直接存储，不按分类组织
│   ├── {resource_id}.{ext}
│   └── ...
└── temp/                  # 临时资源
    ├── {resource_id}.{ext}
```

**注意**: 
- 资源存储路径按用户组织，不包含workflow_id
- 如果workflow需要使用资源，应在workflow模块中维护resource_id列表
- 资源模块不区分分类（input/intermediate/output等），这些由具体的业务模块来决定
- 资源只记录文件扩展名（如：txt、pdf、jpg、mp4等），不维护预定义的类型枚举
- 存储路径简化，直接使用`{user_id}/{resource_id}.{ext}`格式

## 6. 安全考虑

### 6.1 访问控制

- 基于用户权限的资源访问控制（用户只能访问自己的资源）
- 管理员可以访问所有资源
- 预签名URL的过期时间控制
- 资源模块不依赖workflow，访问控制基于用户ID

### 6.2 数据安全

- 文件完整性校验（MD5、SHA256）
- 敏感数据加密存储
- 审计日志记录

### 6.3 存储安全

- 存储后端的安全配置
- 访问密钥的安全管理
- 存储路径的权限控制

## 7. 性能优化

### 7.1 客户端直传

- 减少服务器负载
- 提高上传速度
- 支持大文件上传

### 7.2 缓存策略

- 资源元数据缓存（Redis，5分钟TTL）
- 下载URL缓存（短期）

## 8. 监控和日志

### 8.1 关键指标

- 资源上传数量
- 资源下载次数
- 存储使用量

### 8.2 日志记录

- 资源上传/下载日志
- 错误详情
- 性能指标

## 9. 扩展性设计

### 9.1 存储后端扩展

- 支持新增存储后端（实现Storage接口）
- 配置驱动的存储选择
- 支持多存储后端（主备、分片等）

