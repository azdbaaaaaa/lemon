package resource

import (
	"time"

	"lemon/internal/model/resource"
	httputil "lemon/internal/pkg/http"
)

// ErrorResponse 错误响应类型别名（使用共用的 http.ErrorResponse）
type ErrorResponse = httputil.ErrorResponse

// ResourceInfo 资源信息 DTO
type ResourceInfo struct {
	ID          string                 `json:"id"`                     // 资源ID
	UserID      string                 `json:"user_id"`                // 所属用户ID
	Ext         string                 `json:"ext"`                    // 文件扩展名
	Name        string                 `json:"name"`                   // 原始文件名
	DisplayName string                 `json:"display_name,omitempty"` // 显示名称
	Description string                 `json:"description,omitempty"`  // 描述
	StorageKey  string                 `json:"storage_key"`            // 存储路径
	StorageURL  string                 `json:"storage_url,omitempty"`  // 存储URL
	StorageType string                 `json:"storage_type"`           // 存储类型
	FileSize    int64                  `json:"file_size"`              // 文件大小
	ContentType string                 `json:"content_type"`           // MIME类型
	MD5         string                 `json:"md5,omitempty"`          // MD5值
	SHA256      string                 `json:"sha256,omitempty"`       // SHA256值
	Metadata    map[string]interface{} `json:"metadata,omitempty"`     // 扩展元数据
	Tags        []string               `json:"tags,omitempty"`         // 标签
	Version     int                    `json:"version"`                // 版本号
	ParentID    string                 `json:"parent_id,omitempty"`    // 父资源ID
	Status      string                 `json:"status"`                 // 资源状态
	UploadedAt  string                 `json:"uploaded_at"`            // 上传时间
	CreatedAt   string                 `json:"created_at"`             // 创建时间
	UpdatedAt   string                 `json:"updated_at"`             // 更新时间
}

// toResourceInfo 将 Resource 实体转换为 ResourceInfo DTO
func toResourceInfo(res *resource.Resource) ResourceInfo {
	info := ResourceInfo{
		ID:          res.ID,
		UserID:      res.UserID,
		Ext:         res.Ext,
		Name:        res.Name,
		StorageKey:  res.StorageKey,
		StorageType: res.StorageType,
		FileSize:    res.FileSize,
		ContentType: res.ContentType,
		Version:     res.Version,
		Status:      string(res.Status),
		UploadedAt:  res.UploadedAt.Format(time.RFC3339),
		CreatedAt:   res.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   res.UpdatedAt.Format(time.RFC3339),
	}

	if res.DisplayName != "" {
		info.DisplayName = res.DisplayName
	}
	if res.Description != "" {
		info.Description = res.Description
	}
	if res.StorageURL != "" {
		info.StorageURL = res.StorageURL
	}
	if res.MD5 != "" {
		info.MD5 = res.MD5
	}
	if res.SHA256 != "" {
		info.SHA256 = res.SHA256
	}
	if len(res.Metadata) > 0 {
		info.Metadata = res.Metadata
	}
	if len(res.Tags) > 0 {
		info.Tags = res.Tags
	}
	if res.ParentID != "" {
		info.ParentID = res.ParentID
	}

	return info
}

// toResourceInfoList 将 Resource 实体列表转换为 ResourceInfo DTO 列表
func toResourceInfoList(resources []*resource.Resource) []ResourceInfo {
	list := make([]ResourceInfo, len(resources))
	for i, res := range resources {
		list[i] = toResourceInfo(res)
	}
	return list
}
