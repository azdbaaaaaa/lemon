package resource

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Resource 资源实体
// 资源模块是独立的，不依赖workflow。如果需要在workflow中使用资源，应在workflow模块中维护resource_id关联关系。
// 资源模块不区分分类（input/intermediate/output等），这些由具体的业务模块来决定。
// 资源类型只记录文件扩展名（如：txt、pdf、jpg、mp4等），不维护预定义的类型枚举。
type Resource struct {
	ID          string `bson:"id" json:"id"`                                         // 资源ID（UUID）
	UserID      string `bson:"user_id" json:"user_id"`                               // 所属用户ID
	Ext         string `bson:"ext" json:"ext"`                                       // 文件扩展名（不含点号，如：txt、pdf、jpg、mp4等）
	Name        string `bson:"name" json:"name"`                                     // 原始文件名
	DisplayName string `bson:"display_name,omitempty" json:"display_name,omitempty"` // 显示名称
	Description string `bson:"description,omitempty" json:"description,omitempty"`   // 描述

	// 存储信息
	StorageKey  string `bson:"storage_key" json:"storage_key"`                     // 存储路径（key）
	StorageURL  string `bson:"storage_url,omitempty" json:"storage_url,omitempty"` // 存储URL（临时访问）
	StorageType string `bson:"storage_type" json:"storage_type"`                   // 存储类型（local/oss/s3/minio）

	// 文件信息
	FileSize    int64  `bson:"file_size" json:"file_size"`               // 文件大小（字节）
	ContentType string `bson:"content_type" json:"content_type"`         // MIME类型
	MD5         string `bson:"md5,omitempty" json:"md5,omitempty"`       // 文件MD5值（用于去重）
	SHA256      string `bson:"sha256,omitempty" json:"sha256,omitempty"` // 文件SHA256值

	// 元数据
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"` // 扩展元数据
	Tags     []string               `bson:"tags,omitempty" json:"tags,omitempty"`         // 标签

	// 版本管理
	Version  int    `bson:"version" json:"version"`                         // 版本号
	ParentID string `bson:"parent_id,omitempty" json:"parent_id,omitempty"` // 父资源ID（用于版本链）

	// 状态
	Status ResourceStatus `bson:"status" json:"status"` // 资源状态

	// 时间戳
	UploadedAt time.Time  `bson:"uploaded_at" json:"uploaded_at"`                   // 上传时间
	CreatedAt  time.Time  `bson:"created_at" json:"created_at"`                     // 创建时间
	UpdatedAt  time.Time  `bson:"updated_at" json:"updated_at"`                     // 更新时间
	DeletedAt  *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"` // 软删除时间
}

// ResourceStatus 资源状态
type ResourceStatus string

const (
	ResourceStatusUploading ResourceStatus = "uploading" // 上传中
	ResourceStatusUploaded  ResourceStatus = "uploaded"  // 已上传
	ResourceStatusReady     ResourceStatus = "ready"     // 就绪（可用）
	ResourceStatusFailed    ResourceStatus = "failed"    // 失败
	ResourceStatusDeleted   ResourceStatus = "deleted"   // 已删除
)

// UploadSession 上传会话（用于客户端直传）
type UploadSession struct {
	ID     string `bson:"id" json:"id"`           // 会话ID（UUID）
	UserID string `bson:"user_id" json:"user_id"` // 所属用户ID

	// 文件信息
	FileName    string `bson:"file_name" json:"file_name"`
	FileSize    int64  `bson:"file_size" json:"file_size"`
	ContentType string `bson:"content_type" json:"content_type"`
	Ext         string `bson:"ext" json:"ext"` // 文件扩展名（不含点号，如：txt、pdf、jpg、mp4等）

	// 客户端上传相关
	UploadURL string    `bson:"upload_url" json:"upload_url"` // 客户端上传URL（预签名URL）
	UploadKey string    `bson:"upload_key" json:"upload_key"` // 上传路径（key）
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"` // 上传URL过期时间

	// 上传状态
	Status        UploadStatus `bson:"status" json:"status"`                               // 上传状态
	UploadedBytes int64        `bson:"uploaded_bytes" json:"uploaded_bytes"`               // 已上传字节数
	ResourceID    string       `bson:"resource_id,omitempty" json:"resource_id,omitempty"` // 上传完成后的资源ID

	// 时间戳
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
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

// Collection 返回集合名称
func (r *Resource) Collection() string {
	return "resources"
}

// EnsureIndexes 创建和维护索引
// 参考: docs/design/workflow/RESOURCE_MODULE_DESIGN.md - 2.3 数据库索引
// 注意: 资源模块是独立的，不依赖workflow，也不区分分类（category），这些由业务模块决定
func (r *Resource) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(r.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}, bson.E{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_user_status"),
		},
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}, bson.E{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_user_created"),
		},
		{
			Keys:    bson.D{bson.E{Key: "md5", Value: 1}},
			Options: options.Index().SetName("idx_md5"),
		},
		{
			Keys:    bson.D{bson.E{Key: "storage_key", Value: 1}},
			Options: options.Index().SetName("idx_storage_key"),
		},
		{
			Keys:    bson.D{bson.E{Key: "ext", Value: 1}},
			Options: options.Index().SetName("idx_ext"),
		},
	}

	if len(indexes) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}

// UploadSessionCollection 返回上传会话集合名称
func (s *UploadSession) Collection() string {
	return "upload_sessions"
}

// EnsureIndexes 创建和维护上传会话索引
func (s *UploadSession) EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(s.Collection())
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "user_id", Value: 1}, bson.E{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_user_status"),
		},
		{
			Keys:    bson.D{bson.E{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("idx_expires_at").SetExpireAfterSeconds(0), // TTL索引，自动删除过期会话
		},
	}

	if len(indexes) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
