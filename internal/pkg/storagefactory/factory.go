package storagefactory

import (
	"context"
	"fmt"

	"lemon/internal/config"
	"lemon/internal/pkg/storage"
	"lemon/internal/pkg/storage/local"
	"lemon/internal/pkg/storage/oss"
)

// NewStorage 根据配置创建存储实例
func NewStorage(ctx context.Context, cfg *config.StorageConfig) (storage.Storage, error) {
	switch cfg.Type {
	case "local":
		if cfg.Local == nil {
			return nil, fmt.Errorf("local storage config is required")
		}
		return local.NewLocalStorage(
			cfg.Local.BasePath,
			cfg.Local.BaseURL,
			cfg.Local.PresignExpiry,
		)
	case "oss":
		if cfg.OSS == nil {
			return nil, fmt.Errorf("OSS storage config is required")
		}
		return oss.NewOSSStorage(
			cfg.OSS.Endpoint,
			cfg.OSS.Bucket,
			cfg.OSS.AccessKeyID,
			cfg.OSS.AccessKeySecret,
			cfg.OSS.PresignExpiry,
		)
	case "s3":
		return nil, fmt.Errorf("S3 storage not implemented yet")
	case "minio":
		return nil, fmt.Errorf("MinIO storage not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}
