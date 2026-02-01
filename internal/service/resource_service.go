package service

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

// ResourceService 资源服务接口
// 定义 resource 模块 service 层提供的能力
type ResourceService interface {
	// PrepareUpload 准备上传（创建上传会话）
	// 生成预签名URL供客户端直传
	PrepareUpload(ctx context.Context, req *PrepareUploadRequest) (*PrepareUploadResult, error)

	// CompleteUpload 完成上传（确认上传完成）
	// 客户端上传完成后，验证文件并创建资源记录
	CompleteUpload(ctx context.Context, req *CompleteUploadRequest) (*CompleteUploadResult, error)

	// GetDownloadURL 获取下载URL（预签名URL）
	// 用于生成临时访问链接，适用于客户端直接下载
	// 注意：如果 req.UserID 为空，视为系统内部请求，可以访问所有资源
	GetDownloadURL(ctx context.Context, req *GetDownloadURLRequest) (*GetDownloadURLResult, error)

	// UploadFile 服务端直接上传文件（不通过上传会话）
	// 用于服务端生成的文件（如音频、字幕等）直接上传
	UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResult, error)

	// DownloadFile 下载文件（返回文件流）
	// 用于服务端需要读取文件内容的场景
	// 注意：如果 req.UserID 为空，视为系统内部请求，可以访问所有资源
	DownloadFile(ctx context.Context, req *DownloadFileRequest) (*DownloadFileResult, error)

	// ListResources 查询资源列表
	// 支持按用户ID、扩展名、状态等条件筛选
	// 注意：如果 req.UserID 为空，视为系统内部请求，可以查询所有用户的资源
	ListResources(ctx context.Context, req *ListResourcesRequest) (*ListResourcesResult, error)

	// GetResource 获取资源元数据（不下载文件）
	// 用于查看资源信息、权限验证等场景
	// 注意：如果 req.UserID 为空，视为系统内部请求，可以访问所有资源
	GetResource(ctx context.Context, req *GetResourceRequest) (*GetResourceResult, error)
}

// resourceService 资源服务实现
type resourceService struct {
	resourceRepo *resourceRepo.ResourceRepo
	storage      storage.Storage
}

// NewResourceService 创建资源服务
func NewResourceService(
	resourceRepo *resourceRepo.ResourceRepo,
	storage storage.Storage,
) ResourceService {
	return &resourceService{
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
func (s *resourceService) PrepareUpload(ctx context.Context, req *PrepareUploadRequest) (*PrepareUploadResult, error) {
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
func (s *resourceService) CompleteUpload(ctx context.Context, req *CompleteUploadRequest) (*CompleteUploadResult, error) {
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
func (s *resourceService) createOriginalResource(ctx context.Context, req *CompleteUploadRequest) (*resource.Resource, error) {
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
func (s *resourceService) processResourceChain(ctx context.Context, resourceID string) {
	// TODO: 在需要时，这里可以显式调用脱敏、章节切分等纯函数工具。
	_ = ctx
	_ = resourceID
}

// GetDownloadURLRequest 获取下载URL请求
type GetDownloadURLRequest struct {
	UserID     string        // 用户ID（用于权限验证，为空时视为系统内部请求，可访问所有资源）
	ResourceID string        // 资源ID
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
func (s *resourceService) GetDownloadURL(ctx context.Context, req *GetDownloadURLRequest) (*GetDownloadURLResult, error) {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, req.ResourceID)
	if err != nil {
		return nil, ErrResourceNotFound
	}

	// 检查访问权限（用户只能访问自己的资源）
	// 如果 userID 为空，视为系统内部请求，跳过权限检查
	if req.UserID != "" && res.UserID != req.UserID {
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

// UploadFileRequest 服务端上传文件请求
type UploadFileRequest struct {
	UserID      string
	FileName    string
	ContentType string
	Ext         string // 文件扩展名（不含点号）
	Data        io.Reader
}

// UploadFileResult 服务端上传文件结果
type UploadFileResult struct {
	ResourceID  string `json:"resource_id"`
	ResourceURL string `json:"resource_url"`
	FileSize    int64  `json:"file_size"`
}

// UploadFile 服务端直接上传文件（不通过上传会话）
// 用于服务端生成的文件（如音频、字幕等）直接上传
func (s *resourceService) UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResult, error) {
	if req.Data == nil {
		return nil, errors.New("文件数据不能为空")
	}

	// 读取文件数据并计算哈希
	dataBytes, err := io.ReadAll(req.Data)
	if err != nil {
		return nil, fmt.Errorf("读取文件数据失败: %w", err)
	}

	fileSize := int64(len(dataBytes))

	// 计算 MD5 和 SHA256
	md5Hash := md5.Sum(dataBytes)
	sha256Hash := sha256.Sum256(dataBytes)
	md5Str := hex.EncodeToString(md5Hash[:])
	sha256Str := hex.EncodeToString(sha256Hash[:])

	// 生成资源ID和存储路径
	resourceID := id.New()
	storageKey := s.generateStorageKey(req.UserID, resourceID, req.Ext)

	// 上传文件到存储
	dataReader := strings.NewReader(string(dataBytes))
	_, err = s.storage.Upload(ctx, storageKey, dataReader, req.ContentType)
	if err != nil {
		log.Error().Err(err).Str("key", storageKey).Msg("failed to upload file")
		return nil, errors.New("上传文件失败")
	}

	// 创建资源记录
	res := &resource.Resource{
		ID:          resourceID,
		UserID:      req.UserID,
		Ext:         req.Ext,
		Name:        req.FileName,
		StorageKey:  storageKey,
		StorageType: s.storage.GetStorageType(),
		FileSize:    fileSize,
		ContentType: req.ContentType,
		MD5:         md5Str,
		SHA256:      sha256Str,
		Version:     1,
		Status:      resource.ResourceStatusReady,
	}

	if err := s.resourceRepo.Create(ctx, res); err != nil {
		log.Error().Err(err).Msg("failed to create resource")
		return nil, errors.New("创建资源记录失败")
	}

	// 生成资源访问URL
	resourceURL, err := s.storage.GetPresignedDownloadURL(ctx, storageKey, time.Hour*24)
	if err != nil {
		log.Warn().Err(err).Msg("failed to generate resource URL")
		resourceURL = ""
	}

	return &UploadFileResult{
		ResourceID:  resourceID,
		ResourceURL: resourceURL,
		FileSize:    fileSize,
	}, nil
}

// DownloadFileRequest 下载文件请求
type DownloadFileRequest struct {
	UserID     string // 用户ID（用于权限验证，为空时视为系统内部请求，可访问所有资源）
	ResourceID string // 资源ID
}

// DownloadFileResult 下载文件结果
type DownloadFileResult struct {
	ResourceID  string        `json:"resource_id"`
	FileName    string        `json:"file_name"`
	ContentType string        `json:"content_type"`
	FileSize    int64         `json:"file_size"`
	Data        io.ReadCloser `json:"-"` // 不序列化到JSON
}

// DownloadFile 下载文件（返回文件流）
func (s *resourceService) DownloadFile(ctx context.Context, req *DownloadFileRequest) (*DownloadFileResult, error) {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, req.ResourceID)
	if err != nil {
		return nil, ErrResourceNotFound
	}

	// 检查访问权限
	// 如果 userID 为空，视为系统内部请求，跳过权限检查
	if req.UserID != "" && res.UserID != req.UserID {
		return nil, ErrResourceAccessDenied
	}

	// 检查资源状态
	if res.Status == resource.ResourceStatusDeleted {
		return nil, ErrResourceNotFound
	}

	// 从存储下载文件
	reader, err := s.storage.Download(ctx, res.StorageKey)
	if err != nil {
		log.Error().Err(err).Str("key", res.StorageKey).Msg("failed to download file")
		return nil, errors.New("下载文件失败")
	}

	return &DownloadFileResult{
		ResourceID:  res.ID,
		FileName:    res.Name,
		ContentType: res.ContentType,
		FileSize:    res.FileSize,
		Data:        reader,
	}, nil
}

// ListResourcesRequest 查询资源列表请求
type ListResourcesRequest struct {
	UserID   string // 用户ID（为空时视为系统内部请求，可查询所有用户的资源）
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
func (s *resourceService) ListResources(ctx context.Context, req *ListResourcesRequest) (*ListResourcesResult, error) {
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
	// 如果 userID 为空，视为系统内部请求，查询所有用户的资源
	var resources []*resource.Resource
	var total int64
	var err error

	if req.UserID == "" {
		// 系统内部请求：查询所有资源
		resources, total, err = s.resourceRepo.FindAll(ctx, req.PageSize, offset)
	} else {
		// 普通用户请求：只查询该用户的资源
		resources, total, err = s.resourceRepo.FindByUserID(ctx, req.UserID, req.PageSize, offset)
	}

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

	return &ListResourcesResult{
		Resources: filteredResources,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// GetResourceRequest 获取资源元数据请求
type GetResourceRequest struct {
	UserID     string // 用户ID（用于权限验证，为空时视为系统内部请求，可访问所有资源）
	ResourceID string // 资源ID
}

// GetResourceResult 获取资源元数据结果
type GetResourceResult struct {
	Resource *resource.Resource `json:"resource"`
}

// GetResource 获取资源元数据（不下载文件）
func (s *resourceService) GetResource(ctx context.Context, req *GetResourceRequest) (*GetResourceResult, error) {
	// 查找资源
	res, err := s.resourceRepo.FindByID(ctx, req.ResourceID)
	if err != nil {
		return nil, ErrResourceNotFound
	}

	// 检查访问权限
	// 如果 userID 为空，视为系统内部请求，跳过权限检查
	if req.UserID != "" && res.UserID != req.UserID {
		return nil, ErrResourceAccessDenied
	}

	// 检查资源状态
	if res.Status == resource.ResourceStatusDeleted {
		return nil, ErrResourceNotFound
	}

	return &GetResourceResult{
		Resource: res,
	}, nil
}

// generateStorageKey 生成存储路径
// 格式：resources/{user_id}/{resource_id}.{ext}
func (s *resourceService) generateStorageKey(userID, resourceID, ext string) string {
	if ext != "" {
		return fmt.Sprintf("resources/%s/%s.%s", userID, resourceID, ext)
	}
	return fmt.Sprintf("resources/%s/%s", userID, resourceID)
}
