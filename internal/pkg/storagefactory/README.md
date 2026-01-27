# Storage Factory

存储工厂用于根据配置创建存储实例。

## 使用方法

```go
import (
    "context"
    "lemon/internal/config"
    "lemon/internal/pkg/storagefactory"
)

// 从配置创建存储实例
storage, err := storagefactory.NewStorage(ctx, &cfg.Storage)
if err != nil {
    return err
}

// 使用存储实例
url, err := storage.GetPresignedUploadURL(ctx, "path/to/file.txt", "text/plain", time.Hour)
```

## 支持的存储类型

- `local`: 本地文件系统
- `oss`: 阿里云OSS
- `s3`: AWS S3（未实现）
- `minio`: MinIO（未实现）

## 配置示例

### 本地文件系统

```yaml
storage:
  type: "local"
  local:
    base_path: "./storage"
    base_url: "http://localhost:7080/storage"
    presign_expiry: 3600
```

### 阿里云OSS

```yaml
storage:
  type: "oss"
  oss:
    endpoint: "oss-cn-hangzhou.aliyuncs.com"
    bucket: "your-bucket-name"
    access_key_id: "your-access-key-id"
    access_key_secret: "your-access-key-secret"
    presign_expiry: 3600
```
