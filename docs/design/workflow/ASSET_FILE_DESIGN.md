# 资产文件管理模块设计文档 (DSPEC)

## 1. 模块概述

### 1.1 功能定位

资产文件管理模块负责工作流中所有文件的上传、下载、存储和管理。该模块支持多种文件类型，包括：
- **输入文件**: 小说文档（TXT、PDF、Word等）
- **中间文件**: JSON配置文件、TXT文本文件等
- **媒体文件**: 图片、视频、音频、字幕文件等

### 1.2 核心特性

- **客户端直传**: 采用客户端上传方案，减少服务器负载
- **多文件类型支持**: 支持文档、图片、视频、音频等多种格式
- **文件分类管理**: 按工作流、文件类型、用途进行分类管理
- **访问控制**: 基于用户权限和工作流所有权的访问控制
- **多存储支持**: 支持AWS S3、阿里云OSS、MinIO、本地文件系统等多种存储后端
- **可插拔架构**: 根据配置自动选择存储实现，易于扩展

### 1.3 与其他模块的关系

```
工作流模块
    ↓
资产文件管理模块 ← → 存储服务（S3/本地）
    ↓
剧本生成模块（使用输入文件）
分镜生成模块（使用图片文件）
视频生成模块（使用视频/音频文件）
```

## 2. 数据库结构设计

### 2.1 AssetFile 实体

```go
// internal/model/asset_file.go
package model

import (
    "go.mongodb.org/mongo-driver/bson/primitive"
    "time"
)

// AssetFile 资产文件实体
type AssetFile struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    WorkflowID  string             `bson:"workflow_id" json:"workflow_id"`           // 所属工作流ID
    UserID      string             `bson:"user_id" json:"user_id"`                   // 上传用户ID
    Category    FileCategory       `bson:"category" json:"category"`                 // 文件分类
    Type        FileType           `bson:"type" json:"type"`                         // 文件类型
    Name        string             `bson:"name" json:"name"`                         // 原始文件名
    DisplayName string             `bson:"display_name,omitempty" json:"display_name,omitempty"` // 显示名称
    StorageKey  string             `bson:"storage_key" json:"storage_key"`           // 存储路径（key）
    StorageURL  string             `bson:"storage_url,omitempty" json:"storage_url,omitempty"` // 存储URL（临时访问）
    FileSize    int64              `bson:"file_size" json:"file_size"`               // 文件大小（字节）
    ContentType string             `bson:"content_type" json:"content_type"`         // MIME类型
    MD5         string             `bson:"md5,omitempty" json:"md5,omitempty"`      // 文件MD5值（用于去重）
    SHA256      string             `bson:"sha256,omitempty" json:"sha256,omitempty"` // 文件SHA256值
    Metadata    map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"` // 扩展元数据
    Status      FileStatus         `bson:"status" json:"status"`                     // 文件状态
    UploadedAt  time.Time          `bson:"uploaded_at" json:"uploaded_at"`          // 上传时间
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`            // 创建时间
    UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`            // 更新时间
    DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"` // 软删除时间
}

// FileCategory 文件分类
type FileCategory string

const (
    FileCategoryInput      FileCategory = "input"      // 输入文件（用户上传的原始文件）
    FileCategoryIntermediate FileCategory = "intermediate" // 中间文件（系统生成的中间产物）
    FileCategoryOutput     FileCategory = "output"     // 输出文件（最终生成的文件）
    FileCategoryTemporary  FileCategory = "temporary"  // 临时文件
)

// FileType 文件类型
type FileType string

const (
    // 文档类型
    FileTypeDocument FileType = "document" // 文档（TXT、PDF、Word等）
    FileTypeText     FileType = "text"     // 纯文本文件
    FileTypeJSON     FileType = "json"     // JSON文件
    
    // 图片类型
    FileTypeImage    FileType = "image"    // 图片（JPG、PNG、WEBP等）
    
    // 视频类型
    FileTypeVideo    FileType = "video"    // 视频（MP4、MOV等）
    
    // 音频类型
    FileTypeAudio    FileType = "audio"    // 音频（MP3、WAV等）
    
    // 字幕类型
    FileTypeSubtitle FileType = "subtitle" // 字幕（ASS、SRT等）
)

// FileStatus 文件状态
type FileStatus string

const (
    FileStatusUploading   FileStatus = "uploading"   // 上传中
    FileStatusUploaded    FileStatus = "uploaded"    // 已上传
    FileStatusProcessing  FileStatus = "processing"  // 处理中
    FileStatusReady       FileStatus = "ready"       // 就绪
    FileStatusFailed      FileStatus = "failed"      // 失败
    FileStatusDeleted     FileStatus = "deleted"    // 已删除
)
```

### 2.2 UploadSession 实体（客户端上传会话）

```go
// UploadSession 上传会话（用于客户端直传）
type UploadSession struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    WorkflowID  string             `bson:"workflow_id" json:"workflow_id"`
    UserID      string             `bson:"user_id" json:"user_id"`
    FileName    string             `bson:"file_name" json:"file_name"`
    FileSize    int64              `bson:"file_size" json:"file_size"`
    ContentType string             `bson:"content_type" json:"content_type"`
    Category    FileCategory       `bson:"category" json:"category"`
    Type        FileType           `bson:"type" json:"type"`
    
    // 客户端上传相关
    UploadURL   string             `bson:"upload_url" json:"upload_url"`           // 客户端上传URL（预签名URL）
    UploadKey   string             `bson:"upload_key" json:"upload_key"`           // 上传路径（key）
    ExpiresAt   time.Time          `bson:"expires_at" json:"expires_at"`           // 上传URL过期时间
    
    // 上传状态
    Status      UploadStatus       `bson:"status" json:"status"`                   // 上传状态
    UploadedBytes int64            `bson:"uploaded_bytes" json:"uploaded_bytes"`   // 已上传字节数
    AssetFileID *string            `bson:"asset_file_id,omitempty" json:"asset_file_id,omitempty"` // 上传完成后的文件ID
    
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
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
// 索引定义
// 1. 工作流ID + 分类 + 状态（常用查询）
db.asset_files.createIndex({ workflow_id: 1, category: 1, status: 1 })

// 2. 用户ID + 创建时间（用户文件列表）
db.asset_files.createIndex({ user_id: 1, created_at: -1 })

// 3. MD5值（去重查询）
db.asset_files.createIndex({ md5: 1 })

// 4. 存储路径（快速查找）
db.asset_files.createIndex({ storage_key: 1 })

// 5. 上传会话 - 工作流ID + 状态
db.upload_sessions.createIndex({ workflow_id: 1, status: 1 })

// 6. 上传会话 - 过期时间（清理过期会话）
db.upload_sessions.createIndex({ expires_at: 1 }, { expireAfterSeconds: 0 })
```

## 3. API接口设计

### 3.1 获取上传凭证（客户端直传）

**POST** `/api/v1/workflow/files/upload/prepare`

创建上传会话，返回客户端直传URL。

**请求体**:
```json
{
  "workflow_id": "507f1f77bcf86cd799439011",
  "file_name": "novel.txt",
  "file_size": 1024000,
  "content_type": "text/plain",
  "category": "input",
  "type": "document"
}
```

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "session_001",
    "upload_url": "https://storage.example.com/upload?signature=...",
    "upload_key": "workflows/001/input/novel.txt",
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

**POST** `/api/v1/workflow/files/upload/complete`

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
    "file_id": "file_001",
    "file_url": "https://storage.example.com/workflows/001/input/novel.txt",
    "file_size": 1024000
  }
}
```

### 3.3 查询文件列表

**GET** `/api/v1/workflow/files`

查询工作流的文件列表。

**查询参数**:
- `workflow_id`: 工作流ID（必需）
- `category`: 文件分类筛选（input/intermediate/output）
- `type`: 文件类型筛选（document/image/video/audio等）
- `status`: 状态筛选
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认20）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "files": [
      {
        "id": "file_001",
        "workflow_id": "workflow_001",
        "category": "input",
        "type": "document",
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

**GET** `/api/v1/workflow/files/download`

获取文件下载URL（预签名URL）。

**查询参数**:
- `file_id`: 文件ID（必需）
- `expires_in`: URL有效期（秒，默认3600）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "file_id": "file_001",
    "download_url": "https://storage.example.com/workflows/001/input/novel.txt?signature=...",
    "expires_at": "2024-01-01T13:00:00Z",
    "file_name": "novel.txt",
    "file_size": 1024000,
    "content_type": "text/plain"
  }
}
```

### 3.5 删除文件

**POST** `/api/v1/workflow/files/delete`

删除文件（软删除）。

**请求体**:
```json
{
  "file_id": "file_001"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "删除成功"
}
```

### 3.6 查询文件详情

**GET** `/api/v1/workflow/files`

查询文件详情。

**查询参数**:
- `file_id`: 文件ID（必需）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "file_001",
    "workflow_id": "workflow_001",
    "category": "input",
    "type": "document",
    "name": "novel.txt",
    "display_name": "小说文档",
    "storage_key": "workflows/001/input/novel.txt",
    "file_size": 1024000,
    "content_type": "text/plain",
    "md5": "d41d8cd98f00b204e9800998ecf8427e",
    "status": "ready",
    "uploaded_at": "2024-01-01T10:00:00Z",
    "created_at": "2024-01-01T10:00:00Z",
    "metadata": {
      "chapter_count": 10,
      "word_count": 50000
    }
  }
}
```

## 4. 分层架构设计

### 4.1 Handler层（接口层）

**文件**: `internal/handler/asset_file.go`

**职责**:
- HTTP请求解析和参数校验
- 响应封装
- 权限校验（调用Service层）

**主要方法**:
```go
type AssetFileHandler struct {
    service *service.AssetFileService
}

// PrepareUpload 准备上传（获取上传凭证）
func (h *AssetFileHandler) PrepareUpload(c *gin.Context)

// CompleteUpload 确认上传完成
func (h *AssetFileHandler) CompleteUpload(c *gin.Context)

// ListFiles 查询文件列表
func (h *AssetFileHandler) ListFiles(c *gin.Context)

// GetDownloadURL 获取下载URL
func (h *AssetFileHandler) GetDownloadURL(c *gin.Context)

// DeleteFile 删除文件
func (h *AssetFileHandler) DeleteFile(c *gin.Context)

// GetFileDetail 查询文件详情
func (h *AssetFileHandler) GetFileDetail(c *gin.Context)
```

### 4.2 Service层（业务逻辑层）

**文件**: `internal/service/asset_file_service.go`

**职责**:
- 业务逻辑处理
- 文件类型验证
- 权限校验
- 调用Repository和Storage

**主要方法**:
```go
type AssetFileService struct {
    repo    *repository.AssetFileRepo
    storage storage.Storage
    config  *config.Config
}

// PrepareUpload 准备上传
func (s *AssetFileService) PrepareUpload(ctx context.Context, req *PrepareUploadRequest) (*PrepareUploadResponse, error)

// CompleteUpload 确认上传完成
func (s *AssetFileService) CompleteUpload(ctx context.Context, req *CompleteUploadRequest) (*CompleteUploadResponse, error)

// ListFiles 查询文件列表
func (s *AssetFileService) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error)

// GetDownloadURL 获取下载URL
func (s *AssetFileService) GetDownloadURL(ctx context.Context, fileID string, expiresIn time.Duration) (*DownloadURLResponse, error)

// DeleteFile 删除文件
func (s *AssetFileService) DeleteFile(ctx context.Context, fileID string, userID string) error

// GetFileDetail 查询文件详情
func (s *AssetFileService) GetFileDetail(ctx context.Context, fileID string) (*model.AssetFile, error)

// ValidateFileType 验证文件类型
func (s *AssetFileService) ValidateFileType(fileName string, contentType string, fileSize int64) (FileType, error)

// GenerateStorageKey 生成存储路径
func (s *AssetFileService) GenerateStorageKey(workflowID string, category FileCategory, fileName string) string
```

### 4.3 Repository层（数据访问层）

**文件**: `internal/repository/asset_file_repo.go`

**职责**:
- MongoDB数据访问
- CRUD操作
- 查询优化

**主要方法**:
```go
type AssetFileRepo struct {
    db *mongo.Database
}

// Create 创建文件记录
func (r *AssetFileRepo) Create(ctx context.Context, file *model.AssetFile) error

// FindByID 根据ID查询
func (r *AssetFileRepo) FindByID(ctx context.Context, id string) (*model.AssetFile, error)

// FindByWorkflowID 根据工作流ID查询
func (r *AssetFileRepo) FindByWorkflowID(ctx context.Context, workflowID string, filter *FileFilter) ([]*model.AssetFile, int64, error)

// Update 更新文件记录
func (r *AssetFileRepo) Update(ctx context.Context, id string, updates map[string]interface{}) error

// SoftDelete 软删除
func (r *AssetFileRepo) SoftDelete(ctx context.Context, id string) error

// FindByMD5 根据MD5查询（去重）
func (r *AssetFileRepo) FindByMD5(ctx context.Context, md5 string) (*model.AssetFile, error)

// CreateUploadSession 创建上传会话
func (r *AssetFileRepo) CreateUploadSession(ctx context.Context, session *model.UploadSession) error

// FindUploadSession 查询上传会话
func (r *AssetFileRepo) FindUploadSession(ctx context.Context, sessionID string) (*model.UploadSession, error)

// UpdateUploadSession 更新上传会话
func (r *AssetFileRepo) UpdateUploadSession(ctx context.Context, sessionID string, updates map[string]interface{}) error
```

### 4.4 Storage层（存储抽象层）

**文件**: `internal/pkg/storage/storage.go`

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

// StorageType 存储类型
type StorageType string

const (
    StorageTypeS3     StorageType = "s3"      // AWS S3
    StorageTypeOSS    StorageType = "oss"     // 阿里云OSS
    StorageTypeMinIO  StorageType = "minio"    // MinIO
    StorageTypeLocal  StorageType = "local"   // 本地文件系统
)
```

**实现结构**:
```
internal/pkg/storage/
├── storage.go          # 存储接口定义
├── factory.go          # 存储工厂（根据配置创建存储实例）
├── s3/
│   └── s3.go           # AWS S3实现
├── oss/
│   └── oss.go          # 阿里云OSS实现
├── minio/
│   └── minio.go        # MinIO实现
└── local/
    └── local.go       # 本地文件系统实现
```

**存储工厂实现**:
```go
// factory.go
package storage

import (
    "context"
    "fmt"
    "lemon/internal/config"
)

// NewStorage 根据配置创建存储实例
func NewStorage(ctx context.Context, cfg *config.StorageConfig) (Storage, error) {
    switch cfg.Type {
    case string(StorageTypeS3):
        return NewS3Storage(ctx, cfg.S3)
    case string(StorageTypeOSS):
        return NewOSSStorage(ctx, cfg.OSS)
    case string(StorageTypeMinIO):
        return NewMinIOStorage(ctx, cfg.MinIO)
    case string(StorageTypeLocal):
        return NewLocalStorage(ctx, cfg.Local)
    default:
        return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
    }
}
```

**配置结构**:
```go
// internal/config/config.go
type StorageConfig struct {
    Type  string        `yaml:"type"`  // s3, oss, minio, local
    S3    *S3Config     `yaml:"s3,omitempty"`
    OSS   *OSSConfig    `yaml:"oss,omitempty"`
    MinIO *MinIOConfig  `yaml:"minio,omitempty"`
    Local *LocalConfig  `yaml:"local,omitempty"`
}

// S3Config AWS S3配置
type S3Config struct {
    Region          string `yaml:"region"`
    Bucket          string `yaml:"bucket"`
    AccessKeyID     string `yaml:"access_key_id"`
    SecretAccessKey string `yaml:"secret_access_key"`
    Endpoint        string `yaml:"endpoint,omitempty"` // 自定义端点（用于MinIO等）
    UseSSL          bool   `yaml:"use_ssl"`
    PresignExpiry   int    `yaml:"presign_expiry"` // 预签名URL过期时间（秒）
}

// OSSConfig 阿里云OSS配置
type OSSConfig struct {
    Endpoint        string `yaml:"endpoint"`
    Bucket          string `yaml:"bucket"`
    AccessKeyID     string `yaml:"access_key_id"`
    AccessKeySecret string `yaml:"access_key_secret"`
    PresignExpiry   int    `yaml:"presign_expiry"`
}

// MinIOConfig MinIO配置
type MinIOConfig struct {
    Endpoint        string `yaml:"endpoint"`
    Bucket          string `yaml:"bucket"`
    AccessKeyID     string `yaml:"access_key_id"`
    SecretAccessKey string `yaml:"secret_access_key"`
    UseSSL          bool   `yaml:"use_ssl"`
    PresignExpiry   int    `yaml:"presign_expiry"`
}

// LocalConfig 本地文件系统配置
type LocalConfig struct {
    BasePath      string `yaml:"base_path"`      // 基础路径
    BaseURL       string `yaml:"base_url"`       // 基础URL（用于生成访问URL）
    PresignExpiry int    `yaml:"presign_expiry"` // 预签名URL过期时间（秒）
}
```

## 5. 客户端上传方案

### 5.1 流程设计

```
客户端
    ↓
1. 请求上传凭证
    ↓
服务器生成预签名URL
    ↓
2. 返回上传URL和参数
    ↓
客户端直接上传到存储服务（S3/OSS/MinIO等）
    ↓
3. 上传完成后通知服务器
    ↓
服务器验证文件并创建记录
```

### 5.2 实现细节

#### 5.2.1 生成预签名URL

**S3/MinIO实现**:
```go
func (s *S3Storage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error) {
    req, _ := s.client.PutObjectRequest(&s3.PutObjectInput{
        Bucket:      aws.String(s.bucket),
        Key:         aws.String(key),
        ContentType: aws.String(contentType),
    })
    
    url, err := req.Presign(expiresIn)
    if err != nil {
        return "", err
    }
    
    return url, nil
}
```

**本地文件系统实现**:
```go
func (s *LocalStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error) {
    // 生成临时上传token
    token := generateUploadToken(key, expiresIn)
    
    // 返回服务器上传接口URL（带token）
    return fmt.Sprintf("%s/api/v1/internal/files/upload?token=%s", s.baseURL, token), nil
}
```

#### 5.2.2 文件验证

上传完成后，服务器需要验证：
1. **文件完整性**: 验证MD5/SHA256
2. **文件大小**: 验证实际大小与声明大小一致
3. **文件类型**: 验证Content-Type和文件扩展名
4. **权限校验**: 验证用户是否有权限上传到该工作流

## 6. 文件类型和格式支持

### 6.1 支持的文件格式

| 文件类型 | 支持格式 | 最大大小 | 说明 |
|---------|---------|---------|------|
| 文档 | TXT, PDF, DOC, DOCX | 100MB | 小说、文档等输入文件 |
| 文本 | TXT, JSON, XML | 10MB | 中间配置文件 |
| 图片 | JPG, PNG, WEBP, GIF | 50MB | 分镜图、资产设计图 |
| 视频 | MP4, MOV, AVI | 2GB | 视频片段、最终视频 |
| 音频 | MP3, WAV, AAC | 100MB | 旁白、音效 |
| 字幕 | ASS, SRT, VTT | 1MB | 字幕文件 |

### 6.2 文件类型检测

```go
// ValidateFileType 验证文件类型
func (s *AssetFileService) ValidateFileType(fileName string, contentType string, fileSize int64) (FileType, error) {
    // 1. 根据文件扩展名判断
    ext := strings.ToLower(filepath.Ext(fileName))
    
    // 2. 根据Content-Type判断
    // 3. 验证文件大小限制
    // 4. 返回文件类型或错误
}
```

## 7. 存储路径规范

### 7.1 路径结构

```
workflows/{workflow_id}/
├── input/                    # 输入文件
│   ├── documents/           # 文档文件
│   │   └── {file_id}.{ext}
│   └── ...
├── intermediate/            # 中间文件
│   ├── script/             # 剧本相关
│   │   ├── narration_{chapter_id}.xml
│   │   └── validation_report.json
│   ├── assets/             # 资产相关
│   │   └── asset_{asset_id}.json
│   └── ...
├── output/                  # 输出文件
│   ├── images/             # 图片
│   │   └── {file_id}.{ext}
│   ├── videos/             # 视频
│   │   └── {file_id}.{ext}
│   ├── audio/              # 音频
│   │   └── {file_id}.{ext}
│   └── subtitles/          # 字幕
│       └── {file_id}.{ext}
└── temporary/              # 临时文件（定期清理）
    └── ...
```

### 7.2 文件命名规范

- **输入文件**: 保留原始文件名，添加文件ID前缀避免冲突
  - 格式: `{file_id}_{original_name}`
  - 示例: `file_001_novel.txt`

- **中间文件**: 使用语义化命名
  - 格式: `{purpose}_{identifier}.{ext}`
  - 示例: `narration_chapter_001.xml`, `validation_report.json`

- **输出文件**: 使用文件ID命名
  - 格式: `{file_id}.{ext}`
  - 示例: `file_002.jpg`, `file_003.mp4`

## 8. 安全考虑

### 8.1 访问控制

1. **工作流所有权校验**
   - 用户只能访问自己创建的工作流文件
   - 管理员可以访问所有文件

2. **上传权限校验**
   - 验证用户是否有权限向指定工作流上传文件
   - 验证工作流状态（不允许向已完成的工作流上传输入文件）

3. **文件访问控制**
   - 下载URL使用预签名，设置合理的过期时间
   - 记录文件访问日志

### 8.2 文件安全

1. **文件类型验证**
   - 严格验证文件扩展名和Content-Type
   - 禁止上传可执行文件（.exe, .sh等）
   - 对上传的文件进行病毒扫描（可选）

2. **文件大小限制**
   - 根据文件类型设置不同的最大大小限制
   - 防止恶意上传超大文件

3. **文件内容验证**
   - 对图片文件进行格式验证
   - 对文档文件进行内容扫描（防止恶意代码）

### 8.3 数据安全

1. **文件加密**
   - 敏感文件加密存储
   - 传输使用HTTPS

2. **访问日志**
   - 记录所有文件上传、下载操作
   - 记录访问IP、时间等信息

## 9. 注意事项

### 9.1 性能优化

1. **大文件处理**
   - 支持分片上传（大文件）
   - 使用流式处理，避免内存溢出
   - 异步处理文件验证和元数据提取

2. **存储优化**
   - 相同文件去重（基于MD5）
   - 定期清理临时文件
   - 使用CDN加速文件访问

3. **并发控制**
   - 限制单个用户/工作流的并发上传数
   - 使用队列处理文件验证任务

### 9.2 错误处理

1. **上传失败**
   - 记录失败原因
   - 支持重试机制
   - 清理部分上传的文件

2. **存储失败**
   - 记录错误日志
   - 通知管理员
   - 支持存储后端切换（需要重启服务）
   - 不同存储后端的错误处理：
     - S3: 处理AWS SDK错误码
     - OSS: 处理阿里云OSS错误码
     - MinIO: 处理MinIO错误码
     - Local: 处理文件系统错误

3. **文件损坏**
   - 上传后验证文件完整性
   - 定期检查文件完整性
   - 自动修复或标记损坏文件

### 9.3 数据一致性

1. **事务处理**
   - 文件上传和数据库记录创建使用事务
   - 确保数据一致性
   - 存储操作失败时回滚数据库操作

2. **清理机制**
   - 定期清理过期的上传会话
   - 清理软删除的文件（30天后）
   - 清理临时文件
   - 不同存储后端的清理策略：
     - S3/OSS: 使用生命周期策略自动清理
     - MinIO: 定时任务清理
     - Local: 定时任务清理

3. **存储一致性**
   - 确保数据库记录与存储中的文件一致
   - 定期检查孤立文件（存储中存在但数据库无记录）
   - 定期检查丢失文件（数据库有记录但存储中不存在）

### 9.4 监控和告警

1. **关键指标**
   - 文件上传成功率
   - 文件下载次数
   - 存储使用量
   - 上传/下载速度

2. **告警规则**
   - 存储空间不足
   - 上传失败率过高
   - 异常文件访问

## 10. 实现优先级

### Phase 1: 基础功能（高优先级）
- [ ] 数据库模型定义
- [ ] Repository层实现
- [ ] Storage接口和S3实现
- [ ] 客户端上传凭证生成
- [ ] 文件上传完成确认
- [ ] 文件列表查询
- [ ] 文件下载URL生成

### Phase 2: 增强功能（中优先级）
- [ ] 文件类型验证
- [ ] 文件去重（MD5）
- [ ] 文件删除（软删除）
- [ ] 文件详情查询
- [ ] 本地文件系统存储支持

### Phase 3: 优化功能（低优先级）
- [ ] 大文件分片上传
- [ ] 文件访问日志
- [ ] 存储使用量统计
- [ ] 文件完整性检查
- [ ] CDN集成

## 11. 相关文档

- [工作流架构设计](./ARCHITECTURE.md) - 整体架构设计
- [工作流API设计](./API_DESIGN.md) - API接口设计
- [开发规范](../../guide/DEVELOPMENT_RULES.md) - 开发规范和流程
