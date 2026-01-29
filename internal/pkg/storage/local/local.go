package local

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lemon/internal/pkg/storage"
)

// LocalStorage 本地文件系统存储
type LocalStorage struct {
	basePath      string // 基础路径
	baseURL       string // 基础URL（用于生成访问URL）
	presignExpiry int    // 预签名URL过期时间（秒）
}

// NewLocalStorage 创建本地文件系统存储
func NewLocalStorage(basePath, baseURL string, presignExpiry int) (*LocalStorage, error) {
	// 确保基础路径存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalStorage{
		basePath:      basePath,
		baseURL:       strings.TrimSuffix(baseURL, "/"),
		presignExpiry: presignExpiry,
	}, nil
}

// Upload 上传文件（服务端上传）
func (s *LocalStorage) Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	fullPath := filepath.Join(s.basePath, key)

	// 确保目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 写入数据
	if _, err := io.Copy(file, data); err != nil {
		os.Remove(fullPath) // 删除失败的文件
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 返回文件URL
	return s.getFileURL(key), nil
}

// Download 下载文件
func (s *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// GetPresignedUploadURL 获取预签名上传URL（本地文件系统使用服务器上传接口）
func (s *LocalStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error) {
	// 本地文件系统不支持客户端直传，返回服务器上传接口URL
	// 实际实现中，可以通过生成临时token来实现
	token := generateUploadToken(key, expiresIn)
	url := fmt.Sprintf("%s/api/v1/internal/resources/upload?token=%s&key=%s", s.baseURL, token, key)
	return url, nil
}

// GetPresignedDownloadURL 获取预签名下载URL
func (s *LocalStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	// 本地文件系统直接返回文件URL（实际应该生成临时访问token）
	// 简化实现：直接返回文件URL
	return s.getFileURL(key), nil
}

// Delete 删除文件
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，认为删除成功
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists 检查文件是否存在
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(s.basePath, key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetFileInfo 获取文件信息
func (s *LocalStorage) GetFileInfo(ctx context.Context, key string) (*storage.FileInfo, error) {
	fullPath := filepath.Join(s.basePath, key)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// 计算ETag（使用MD5）
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}
	etag := hex.EncodeToString(hash.Sum(nil))

	// 获取Content-Type
	contentType := getContentType(key)

	return &storage.FileInfo{
		Key:          key,
		Size:         info.Size(),
		ContentType:  contentType,
		ETag:         etag,
		LastModified: info.ModTime(),
	}, nil
}

// GetStorageType 获取存储类型
func (s *LocalStorage) GetStorageType() string {
	return string(storage.StorageTypeLocal)
}

// getFileURL 获取文件URL
func (s *LocalStorage) getFileURL(key string) string {
	// 将路径中的反斜杠替换为正斜杠
	urlKey := strings.ReplaceAll(key, "\\", "/")
	return fmt.Sprintf("%s/%s", s.baseURL, urlKey)
}

// generateUploadToken 生成上传token（简化实现）
func generateUploadToken(key string, expiresIn time.Duration) string {
	// 实际实现中应该使用JWT或其他安全方式生成token
	// 这里简化实现，仅用于演示
	expiresAt := time.Now().Add(expiresIn).Unix()
	data := fmt.Sprintf("%s:%d", key, expiresAt)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// getContentType 根据文件扩展名获取Content-Type
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	contentTypes := map[string]string{
		".txt":  "text/plain",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ass":  "text/x-ass",
		".srt":  "text/plain",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
