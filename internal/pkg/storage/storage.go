package storage

import (
	"context"
	"io"
	"time"
)

// Storage 存储接口
type Storage interface {
	// Upload 上传文件（服务端上传）
	Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)

	// Download 下载文件
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// GetPresignedUploadURL 获取预签名上传URL（客户端直传）
	GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error)

	// GetPresignedDownloadURL 获取预签名下载URL
	GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)

	// Delete 删除文件
	Delete(ctx context.Context, key string) error

	// Exists 检查文件是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// GetFileInfo 获取文件信息
	GetFileInfo(ctx context.Context, key string) (*FileInfo, error)

	// GetStorageType 获取存储类型
	GetStorageType() string
}

// FileInfo 文件信息
type FileInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal StorageType = "local" // 本地文件系统
	StorageTypeOSS   StorageType = "oss"   // 阿里云OSS
	StorageTypeS3    StorageType = "s3"    // AWS S3
	StorageTypeMinIO StorageType = "minio" // MinIO
)
