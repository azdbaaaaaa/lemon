package storagefactory

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lemon/internal/config"
)

func getTestTmpDir(t *testing.T) string {
	// 获取项目根目录
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	// 向上找到项目根目录（从 internal/pkg/storagefactory 到项目根）
	for !strings.HasSuffix(projectRoot, "lemon") && len(projectRoot) > 1 {
		projectRoot = filepath.Dir(projectRoot)
	}
	if !strings.HasSuffix(projectRoot, "lemon") {
		t.Fatalf("Failed to find project root")
	}

	// 使用 tmp 目录作为测试存储路径
	return filepath.Join(projectRoot, "tmp", "storage_test")
}

func TestNewStorage_Local(t *testing.T) {
	tmpDir := getTestTmpDir(t)
	baseURL := "http://localhost:8080/storage"

	// 清理测试目录
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		cfg     *config.StorageConfig
		wantErr bool
	}{
		{
			name: "valid local storage config",
			cfg: &config.StorageConfig{
				Type: "local",
				Local: &config.LocalConfig{
					BasePath:      tmpDir,
					BaseURL:       baseURL,
					PresignExpiry: 3600,
				},
			},
			wantErr: true, // local storage 尚未实现，期望返回错误
		},
		{
			name: "missing local config",
			cfg: &config.StorageConfig{
				Type:  "local",
				Local: nil,
			},
			wantErr: true,
		},
		{
			name: "unsupported storage type",
			cfg: &config.StorageConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage, err := NewStorage(ctx, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewStorage() expected error, got nil")
				}
				if storage != nil {
					t.Errorf("NewStorage() expected nil storage, got %v", storage)
				}
				return
			}

			if err != nil {
				t.Errorf("NewStorage() unexpected error: %v", err)
				return
			}

			if storage == nil {
				t.Errorf("NewStorage() expected storage instance, got nil")
				return
			}
		})
	}
}

func TestLocalStorage_Operations(t *testing.T) {
	t.Skip("Local storage 尚未实现，跳过此测试")
	tmpDir := getTestTmpDir(t)
	baseURL := "http://localhost:8080/storage"

	// 清理测试目录
	defer os.RemoveAll(tmpDir)

	cfg := &config.StorageConfig{
		Type: "local",
		Local: &config.LocalConfig{
			BasePath:      tmpDir,
			BaseURL:       baseURL,
			PresignExpiry: 3600,
		},
	}

	ctx := context.Background()
	storage, err := NewStorage(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// 测试上传
	testKey := "test/test.txt"
	testContent := "Hello, World! This is a test file."
	testReader := strings.NewReader(testContent)

	url, err := storage.Upload(ctx, testKey, testReader, "text/plain")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	expectedURL := baseURL + "/" + testKey
	if url != expectedURL {
		t.Errorf("Upload() url = %v, want %v", url, expectedURL)
	}

	// 验证文件是否存在
	exists, err := storage.Exists(ctx, testKey)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Errorf("Exists() = false, want true")
	}

	// 测试下载
	reader, err := storage.Download(ctx, testKey)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer reader.Close()

	downloadedContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(downloadedContent) != testContent {
		t.Errorf("Download() content = %v, want %v", string(downloadedContent), testContent)
	}

	// 测试获取文件信息
	fileInfo, err := storage.GetFileInfo(ctx, testKey)
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}

	if fileInfo.Key != testKey {
		t.Errorf("GetFileInfo() Key = %v, want %v", fileInfo.Key, testKey)
	}

	if fileInfo.Size != int64(len(testContent)) {
		t.Errorf("GetFileInfo() Size = %v, want %v", fileInfo.Size, len(testContent))
	}

	if fileInfo.ContentType != "text/plain" {
		t.Errorf("GetFileInfo() ContentType = %v, want 'text/plain'", fileInfo.ContentType)
	}

	// 测试预签名下载URL
	presignedURL, err := storage.GetPresignedDownloadURL(ctx, testKey, time.Hour)
	if err != nil {
		t.Fatalf("GetPresignedDownloadURL() error = %v", err)
	}

	if presignedURL != expectedURL {
		t.Errorf("GetPresignedDownloadURL() url = %v, want %v", presignedURL, expectedURL)
	}

	// 测试预签名上传URL
	presignedUploadURL, err := storage.GetPresignedUploadURL(ctx, "test/upload.txt", "text/plain", time.Hour)
	if err != nil {
		t.Fatalf("GetPresignedUploadURL() error = %v", err)
	}

	if !strings.Contains(presignedUploadURL, baseURL) {
		t.Errorf("GetPresignedUploadURL() url = %v, should contain %v", presignedUploadURL, baseURL)
	}

	// 测试删除
	err = storage.Delete(ctx, testKey)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 验证文件已删除
	exists, err = storage.Exists(ctx, testKey)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Errorf("Exists() = true, want false (file should be deleted)")
	}
}

func TestLocalStorage_NonExistentFile(t *testing.T) {
	t.Skip("Local storage 尚未实现，跳过此测试")
	tmpDir := getTestTmpDir(t)
	baseURL := "http://localhost:8080/storage"

	// 清理测试目录
	defer os.RemoveAll(tmpDir)

	cfg := &config.StorageConfig{
		Type: "local",
		Local: &config.LocalConfig{
			BasePath:      tmpDir,
			BaseURL:       baseURL,
			PresignExpiry: 3600,
		},
	}

	ctx := context.Background()
	storage, err := NewStorage(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	nonExistentKey := "nonexistent/file.txt"

	// 测试下载不存在的文件
	_, err = storage.Download(ctx, nonExistentKey)
	if err == nil {
		t.Errorf("Download() expected error for non-existent file, got nil")
	}

	// 测试获取不存在的文件信息
	_, err = storage.GetFileInfo(ctx, nonExistentKey)
	if err == nil {
		t.Errorf("GetFileInfo() expected error for non-existent file, got nil")
	}

	// 测试删除不存在的文件（应该成功）
	err = storage.Delete(ctx, nonExistentKey)
	if err != nil {
		t.Errorf("Delete() error = %v, should succeed for non-existent file", err)
	}
}
