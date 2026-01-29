package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/resource"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/storage"
	resourceRepo "lemon/internal/repository/resource"
)

var (
	ErrResourceNotFound      = errors.New("资源不存在")
	ErrResourceAccessDenied  = errors.New("无权访问该资源")
	ErrUploadSessionNotFound = errors.New("上传会话不存在")
	ErrUploadSessionExpired  = errors.New("上传会话已过期")
	ErrUploadSessionInvalid  = errors.New("上传会话状态无效")
	ErrFileNotFound          = errors.New("文件不存在")
	ErrInvalidFileHash       = errors.New("文件哈希值不匹配")
)

// ResourceService 资源服务
type ResourceService struct {
	resourceRepo *resourceRepo.ResourceRepo
	storage      storage.Storage
}

// NewResourceService 创建资源服务
func NewResourceService(
	resourceRepo *resourceRepo.ResourceRepo,
	storage storage.Storage,
) *ResourceService {
	return &ResourceService{
		resourceRepo: resourceRepo,
		storage:      storage,
	}
}

// PrepareUploadRequest 准备上传请求
type PrepareUploadRequest struct {
	UserID      string
	FileName    string
	FileSize    int64
	ContentType string
	Ext         string // 文件扩展名（不含点号）
}

// PrepareUploadResult 准备上传结果
type PrepareUploadResult struct {
	SessionID    string    `json:"session_id"`
	UploadURL    string    `json:"upload_url"`
	UploadKey    string    `json:"upload_key"`
	ExpiresAt    time.Time `json:"expires_at"`
	UploadMethod string    `json:"upload_method"` // PUT 或 POST
}

// PrepareUpload 准备上传（创建上传会话）
// 生成预签名URL供客户端直传
func (s *ResourceService) PrepareUpload(ctx context.Context, req *PrepareUploadRequest) (*PrepareUploadResult, error) {
	// 生成上传会话ID
	sessionID := id.New()

	// 生成存储路径：resources/{user_id}/{resource_id}.{ext}
	// 注意：这里使用 sessionID 作为临时资源ID，上传完成后会创建正式资源
	storageKey := s.generateStorageKey(req.UserID, sessionID, req.Ext)

	// 生成预签名上传URL（有效期1小时）
	expiresIn := time.Hour
	uploadURL, err := s.storage.GetPresignedUploadURL(ctx, storageKey, req.ContentType, expiresIn)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate presigned upload URL")
		return nil, errors.New("生成上传URL失败")
	}

	// 计算过期时间
	expiresAt := time.Now().Add(expiresIn)

	// 创建上传会话
	session := &resource.UploadSession{
		ID:            sessionID,
		UserID:        req.UserID,
		FileName:      req.FileName,
		FileSize:      req.FileSize,
		ContentType:   req.ContentType,
		Ext:           req.Ext,
		UploadURL:     uploadURL,
		UploadKey:     storageKey,
		ExpiresAt:     expiresAt,
		Status:        resource.UploadStatusPending,
		UploadedBytes: 0,
	}

	if err := s.resourceRepo.CreateUploadSession(ctx, session); err != nil {
		log.Error().Err(err).Msg("failed to create upload session")
		return nil, errors.New("创建上传会话失败")
	}

	return &PrepareUploadResult{
		SessionID:    sessionID,
		UploadURL:    uploadURL,
		UploadKey:    storageKey,
		ExpiresAt:    expiresAt,
		UploadMethod: "PUT", // 大多数对象存储使用PUT方法
	}, nil
}

// CompleteUploadRequest 完成上传请求
type CompleteUploadRequest struct {
	SessionID string
	MD5       string
	SHA256    string
}

// CompleteUploadResult 完成上传结果
type CompleteUploadResult struct {
	ResourceID  string `json:"resource_id"`
	ResourceURL string `json:"resource_url"`
	FileSize    int64  `json:"file_size"`
}

// CompleteUpload 完成上传（确认上传完成）
// 客户端上传完成后，验证文件并创建资源记录
func (s *ResourceService) CompleteUpload(ctx context.Context, req *CompleteUploadRequest) (*CompleteUploadResult, error) {
	// 验证会话、保存原始资源并更新会话状态
	originalRes, err := s.createOriginalResource(ctx, req)
	if err != nil {
		return nil, err
	}

	// 生成资源访问URL（使用原始资源）
	resourceURL, err := s.storage.GetPresignedDownloadURL(ctx, originalRes.StorageKey, time.Hour*24)
	if err != nil {
		log.Warn().Err(err).Msg("failed to generate resource URL")
		// 不影响主流程，返回空URL
		resourceURL = ""
	}

	// 异步执行后续处理链（脱敏等处理），不阻塞主流程
	go s.processResourceChain(context.Background(), originalRes.ID)

	return &CompleteUploadResult{
		ResourceID:  originalRes.ID,
		ResourceURL: resourceURL,
		FileSize:    originalRes.FileSize,
	}, nil
}

// createOriginalResource 验证会话并创建原始资源记录
// 创建成功后会自动更新上传会话状态为已完成
func (s *ResourceService) createOriginalResource(ctx context.Context, req *CompleteUploadRequest) (*resource.Resource, error) {
	// 查找上传会话
	session, err := s.resourceRepo.FindUploadSession(ctx, req.SessionID)
	if err != nil {
		return nil, ErrUploadSessionNotFound
	}

	// 检查会话是否过期
	if time.Now().After(session.ExpiresAt) {
		_ = s.resourceRepo.UpdateUploadSession(ctx, req.SessionID, map[string]interface{}{
			"status": resource.UploadStatusExpired,
		})
		return nil, ErrUploadSessionExpired
	}

	// 检查会话状态
	if session.Status != resource.UploadStatusPending && session.Status != resource.UploadStatusUploading {
		return nil, ErrUploadSessionInvalid
	}

	// 验证文件是否存在
	exists, err := s.storage.Exists(ctx, session.UploadKey)
	if err != nil {
		log.Error().Err(err).Str("key", session.UploadKey).Msg("failed to check file existence")
		return nil, errors.New("验证文件失败")
	}
	if !exists {
		return nil, ErrFileNotFound
	}

	// 获取文件信息
	fileInfo, err := s.storage.GetFileInfo(ctx, session.UploadKey)
	if err != nil {
		log.Error().Err(err).Str("key", session.UploadKey).Msg("failed to get file info")
		return nil, errors.New("获取文件信息失败")
	}

	// 验证文件大小
	if fileInfo.Size != session.FileSize {
		log.Warn().
			Int64("expected", session.FileSize).
			Int64("actual", fileInfo.Size).
			Msg("file size mismatch")
		_ = s.resourceRepo.UpdateUploadSession(ctx, req.SessionID, map[string]interface{}{
			"status": resource.UploadStatusFailed,
		})
		return nil, errors.New("文件大小不匹配")
	}

	// 生成资源ID
	resourceID := id.New()

	// 创建原始资源记录（保留原始文件）
	originalRes := &resource.Resource{
		ID:          resourceID,
		UserID:      session.UserID,
		Ext:         session.Ext,
		Name:        session.FileName,
		StorageKey:  session.UploadKey,
		StorageType: s.storage.GetStorageType(),
		FileSize:    fileInfo.Size,
		ContentType: fileInfo.ContentType,
		MD5:         req.MD5,
		SHA256:      req.SHA256,
		Version:     1,
		Status:      resource.ResourceStatusReady,
	}

	// 保存原始资源
	if err := s.resourceRepo.Create(ctx, originalRes); err != nil {
		log.Error().Err(err).Msg("failed to create resource")
		return nil, errors.New("创建资源失败")
	}

	// 更新上传会话状态为已完成（原始资源创建成功后，上传即完成）
	if err := s.resourceRepo.UpdateUploadSession(ctx, req.SessionID, map[string]interface{}{
		"status":         resource.UploadStatusCompleted,
		"resource_id":    resourceID,
		"uploaded_bytes": fileInfo.Size,
	}); err != nil {
		log.Warn().Err(err).Msg("failed to update upload session")
		// 不影响主流程，只记录警告
	}

	log.Info().
		Str("resource_id", resourceID).
		Str("storage_key", session.UploadKey).
		Int64("file_size", fileInfo.Size).
		Msg("原始资源创建成功，上传会话已更新")

	return originalRes, nil
}

// processResourceChain 执行资源处理链（脱敏等处理）
// processResourceChain 原先用于异步执行资源处理链（脱敏、章节切分等）。
// 目前资源处理链已改为在 service 层显式调用纯函数，因此该方法留空或后续重构为具体业务流程。
func (s *ResourceService) processResourceChain(ctx context.Context, resourceID string) {
	// TODO: 在需要时，这里可以显式调用脱敏、章节切分等纯函数工具。
	_ = ctx
	_ = resourceID
}

// GetDownloadURLRequest 获取下载URL请求
type GetDownloadURLRequest struct {
	ResourceID string
	ExpiresIn  time.Duration // 可选，默认1小时
}

// GetDownloadURLResult 获取下载URL结果
type GetDownloadURLResult struct {
	ResourceID  string    `json:"resource_id"`
	DownloadURL string    `json:"download_url"`
	ExpiresAt   time.Time `json:"expires_at"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
}

// GetDownloadURL 获取下载URL（预签名URL）
func (s *ResourceService) GetDownloadURL(ctx context.Context, userID string, req *GetDownloadURLRequest) (*GetDownloadURLResult, error) {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, req.ResourceID)
	if err != nil {
		return nil, ErrResourceNotFound
	}

	// 检查访问权限（用户只能访问自己的资源）
	if res.UserID != userID {
		return nil, ErrResourceAccessDenied
	}

	// 检查资源状态
	if res.Status == resource.ResourceStatusDeleted {
		return nil, ErrResourceNotFound
	}

	// 设置默认过期时间
	expiresIn := req.ExpiresIn
	if expiresIn == 0 {
		expiresIn = time.Hour
	}

	// 生成预签名下载URL
	downloadURL, err := s.storage.GetPresignedDownloadURL(ctx, res.StorageKey, expiresIn)
	if err != nil {
		log.Error().Err(err).Str("key", res.StorageKey).Msg("failed to generate download URL")
		return nil, errors.New("生成下载URL失败")
	}

	return &GetDownloadURLResult{
		ResourceID:  res.ID,
		DownloadURL: downloadURL,
		ExpiresAt:   time.Now().Add(expiresIn),
		FileName:    res.Name,
		FileSize:    res.FileSize,
		ContentType: res.ContentType,
	}, nil
}

// ListResourcesRequest 查询资源列表请求
type ListResourcesRequest struct {
	UserID   string // 用户ID（必需）
	Ext      string // 文件扩展名筛选（可选）
	Status   string // 状态筛选（可选）
	Page     int    // 页码（默认1）
	PageSize int    // 每页数量（默认20）
}

// ListResourcesResult 查询资源列表结果
type ListResourcesResult struct {
	Resources []*resource.Resource `json:"resources"`
	Total     int64                `json:"total"`
	Page      int                  `json:"page"`
	PageSize  int                  `json:"page_size"`
}

// ListResources 查询资源列表
func (s *ResourceService) ListResources(ctx context.Context, req *ListResourcesRequest) (*ListResourcesResult, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大页面大小
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize

	// 查询资源列表
	resources, total, err := s.resourceRepo.FindByUserID(ctx, req.UserID, req.PageSize, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list resources")
		return nil, errors.New("查询资源列表失败")
	}

	// 应用筛选（扩展名和状态）
	filteredResources := make([]*resource.Resource, 0)
	for _, res := range resources {
		// 扩展名筛选
		if req.Ext != "" && res.Ext != req.Ext {
			continue
		}
		// 状态筛选
		if req.Status != "" && string(res.Status) != req.Status {
			continue
		}
		filteredResources = append(filteredResources, res)
	}

	// 注意：这里简化处理，实际应该在数据库层面进行筛选以提高性能
	// 当前实现先查询所有数据再筛选，对于大数据量会有性能问题

	return &ListResourcesResult{
		Resources: filteredResources,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// GetResource 获取资源详情
func (s *ResourceService) GetResource(ctx context.Context, userID string, resourceID string) (*resource.Resource, error) {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		return nil, ErrResourceNotFound
	}

	// 检查访问权限（用户只能访问自己的资源）
	if res.UserID != userID {
		return nil, ErrResourceAccessDenied
	}

	// 检查资源状态
	if res.Status == resource.ResourceStatusDeleted {
		return nil, ErrResourceNotFound
	}

	return res, nil
}

// DeleteResource 删除资源（软删除）
func (s *ResourceService) DeleteResource(ctx context.Context, userID string, resourceID string) error {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		return ErrResourceNotFound
	}

	// 检查访问权限（用户只能删除自己的资源）
	if res.UserID != userID {
		return ErrResourceAccessDenied
	}

	// 执行软删除
	if err := s.resourceRepo.Delete(ctx, resourceID); err != nil {
		log.Error().Err(err).Str("resource_id", resourceID).Msg("failed to delete resource")
		return errors.New("删除资源失败")
	}

	// 注意：这里只做软删除，不删除实际存储的文件
	// 如果需要物理删除，可以在这里调用 storage.Delete
	// 但通常建议保留文件以便恢复，或者通过定时任务清理

	return nil
}

// UpdateResourceRequest 更新资源请求
type UpdateResourceRequest struct {
	DisplayName string                 `json:"display_name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateResource 更新资源信息
func (s *ResourceService) UpdateResource(ctx context.Context, userID string, resourceID string, req *UpdateResourceRequest) (*resource.Resource, error) {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		return nil, ErrResourceNotFound
	}

	// 检查访问权限（用户只能更新自己的资源）
	if res.UserID != userID {
		return nil, ErrResourceAccessDenied
	}

	// 检查资源状态
	if res.Status == resource.ResourceStatusDeleted {
		return nil, ErrResourceNotFound
	}

	// 构建更新数据
	updates := make(map[string]interface{})
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	if len(updates) == 0 {
		return res, nil // 没有需要更新的字段
	}

	// 更新资源
	if err := s.resourceRepo.Update(ctx, resourceID, updates); err != nil {
		log.Error().Err(err).Str("resource_id", resourceID).Msg("failed to update resource")
		return nil, errors.New("更新资源失败")
	}

	// 重新查询资源以返回最新数据
	updatedRes, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		log.Error().Err(err).Str("resource_id", resourceID).Msg("failed to get updated resource")
		return nil, errors.New("获取更新后的资源失败")
	}

	return updatedRes, nil
}

// generateStorageKey 生成存储路径
// 格式：resources/{user_id}/{resource_id}.{ext}
func (s *ResourceService) generateStorageKey(userID, resourceID, ext string) string {
	// 确保扩展名格式正确（不含点号）
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "bin" // 默认扩展名
	}

	// 构建路径
	return fmt.Sprintf("resources/%s/%s.%s", userID, resourceID, ext)
}
