package oss

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"lemon/internal/pkg/storage"
)

// OSSStorage 阿里云OSS存储
type OSSStorage struct {
	bucket        *oss.Bucket
	bucketName    string
	presignExpiry int // 预签名URL过期时间（秒）
}

// NewOSSStorage 创建阿里云OSS存储
func NewOSSStorage(endpoint, bucketName, accessKeyID, accessKeySecret string, presignExpiry int) (*OSSStorage, error) {
	// 创建OSS客户端
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	// 获取Bucket
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &OSSStorage{
		bucket:        bucket,
		bucketName:    bucketName,
		presignExpiry: presignExpiry,
	}, nil
}

// Upload 上传文件（服务端上传）
func (s *OSSStorage) Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	options := []oss.Option{
		oss.ContentType(contentType),
	}

	err := s.bucket.PutObject(key, data, options...)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// 返回文件URL
	url := fmt.Sprintf("https://%s.%s/%s", s.bucketName, s.bucket.Client.Config.Endpoint, key)
	return url, nil
}

// Download 下载文件
func (s *OSSStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	body, err := s.bucket.GetObject(key)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return body, nil
}

// GetPresignedUploadURL 获取预签名上传URL（客户端直传）
func (s *OSSStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error) {
	// 如果配置的过期时间大于请求的过期时间，使用配置的过期时间
	expiry := expiresIn
	if time.Duration(s.presignExpiry)*time.Second < expiresIn {
		expiry = time.Duration(s.presignExpiry) * time.Second
	}

	options := []oss.Option{
		oss.ContentType(contentType),
	}

	url, err := s.bucket.SignURL(key, oss.HTTPPut, int64(expiry.Seconds()), options...)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return url, nil
}

// GetPresignedDownloadURL 获取预签名下载URL
func (s *OSSStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	// 如果配置的过期时间大于请求的过期时间，使用配置的过期时间
	expiry := expiresIn
	if time.Duration(s.presignExpiry)*time.Second < expiresIn {
		expiry = time.Duration(s.presignExpiry) * time.Second
	}

	url, err := s.bucket.SignURL(key, oss.HTTPGet, int64(expiry.Seconds()))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return url, nil
}

// Delete 删除文件
func (s *OSSStorage) Delete(ctx context.Context, key string) error {
	err := s.bucket.DeleteObject(key)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists 检查文件是否存在
func (s *OSSStorage) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := s.bucket.IsObjectExist(key)
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	return exists, nil
}

// GetFileInfo 获取文件信息
func (s *OSSStorage) GetFileInfo(ctx context.Context, key string) (*storage.FileInfo, error) {
	props, err := s.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 解析文件大小
	var size int64
	if sizeStr := props.Get("Content-Length"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	// 获取Content-Type
	contentType := props.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 获取ETag
	etag := props.Get("ETag")
	if etag != "" {
		// 移除ETag的引号
		etag = strings.Trim(etag, `"`)
	}

	// 获取Last-Modified
	var lastModified time.Time
	if lastModifiedStr := props.Get("Last-Modified"); lastModifiedStr != "" {
		lastModified, _ = time.Parse(time.RFC1123, lastModifiedStr)
	}

	return &storage.FileInfo{
		Key:          key,
		Size:         size,
		ContentType:  contentType,
		ETag:         etag,
		LastModified: lastModified,
	}, nil
}

// GetStorageType 获取存储类型
func (s *OSSStorage) GetStorageType() string {
	return string(storage.StorageTypeOSS)
}
