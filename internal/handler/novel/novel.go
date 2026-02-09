package novel

import (
	"net/http"
	"time"

	httputil "lemon/internal/pkg/http"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/ctxutil"

	"github.com/gin-gonic/gin"
)

// NovelInfo 剧本信息 DTO
type NovelInfo struct {
	ID              string `json:"id"`                    // 剧本ID
	ResourceID      string `json:"resource_id"`           // 资源ID
	UserID          string `json:"user_id"`               // 用户ID
	Title           string `json:"title,omitempty"`       // 剧本标题
	Author          string `json:"author,omitempty"`      // 作者
	Description     string `json:"description,omitempty"` // 简介
	NarrationType   string `json:"narration_type"`        // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
	Style           string `json:"style"`                 // 风格：anime（漫剧）、live（真人剧）、mixed（混合）
	EpisodeCount    int    `json:"episode_count"`         // 集数（章节数量）
	EpisodeDuration string `json:"episode_duration"`      // 每集时长：auto（自动）、3-5min、5-10min、10-20min、20-30min
	CreatedAt       string `json:"created_at"`            // 创建时间
	UpdatedAt       string `json:"updated_at"`            // 更新时间
}

// FromNovelEntity 从 Novel 实体创建 NovelInfo
func (info *NovelInfo) FromNovelEntity(novelEntity *novel.Novel) {
	info.ID = novelEntity.ID
	info.ResourceID = novelEntity.ResourceID
	info.UserID = novelEntity.UserID
	info.Title = novelEntity.Title
	info.Author = novelEntity.Author
	info.Description = novelEntity.Description
	info.NarrationType = string(novelEntity.NarrationType)
	info.Style = string(novelEntity.Style)
	info.EpisodeCount = novelEntity.EpisodeCount
	info.EpisodeDuration = string(novelEntity.EpisodeDuration)
	info.CreatedAt = novelEntity.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = novelEntity.UpdatedAt.Format(time.RFC3339)
}

// CreateNovelRequest 创建剧本请求
type CreateNovelRequest struct {
	ResourceID    string `json:"resource_id" binding:"required"`    // 资源ID（必填）
	UserID        string `json:"user_id" binding:"required"`        // 用户ID（必填）
	NarrationType string `json:"narration_type" binding:"required"` // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
	Style         string `json:"style" binding:"required"`          // 风格：anime（漫剧）、live（真人剧）、mixed（混合）
	// 注意：格式和每集时长在切分章节时自动确定
}

// CreateNovelResponseData 创建剧本响应数据
type CreateNovelResponseData struct {
	NovelID string `json:"novel_id"` // 创建的剧本ID
}

// CreateNovel 根据资源ID创建剧本
// @Summary      创建剧本
// @Description  根据资源ID创建剧本，返回剧本ID。这是剧本处理流程的第一步。
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      CreateNovelRequest  true  "创建剧本请求"
// @Success      201      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"剧本创建成功\", \"data\": {\"novel_id\": \"...\"}}"
// @Failure      400      {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500      {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels [post]
func (h *Handler) CreateNovel(c *gin.Context) {
	var req CreateNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 将请求中的字符串类型转换为枚举类型
	var narrationType novel.NarrationType
	switch req.NarrationType {
	case string(novel.NarrationTypeNarration):
		narrationType = novel.NarrationTypeNarration
	case string(novel.NarrationTypeDialogue):
		narrationType = novel.NarrationTypeDialogue
	default:
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40002,
			Message: "invalid narration_type, must be narration or dialogue",
		})
		return
	}

	var style novel.NovelStyle
	switch req.Style {
	case string(novel.NovelStyleAnime):
		style = novel.NovelStyleAnime
	case string(novel.NovelStyleLive):
		style = novel.NovelStyleLive
	case string(novel.NovelStyleMixed):
		style = novel.NovelStyleMixed
	default:
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40003,
			Message: "invalid style, must be anime, live, or mixed",
		})
		return
	}

	// 调用Service层（格式和每集时长在切分章节时自动确定）
	novelID, err := h.novelService.CreateNovelFromResource(ctx, req.ResourceID, req.UserID, narrationType, style)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "failed to find resource" {
			code = http.StatusBadRequest
			errorCode = 40004
		}

		c.JSON(code, httputil.ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "剧本创建成功",
		"data": CreateNovelResponseData{
			NovelID: novelID,
		},
	})
}

// GetNovelRequest 获取剧本请求
type GetNovelRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 剧本ID（必填）
}

// GetNovelResponseData 获取剧本响应数据
type GetNovelResponseData struct {
	Novel NovelInfo `json:"novel"` // 剧本信息
}

// GetNovel 获取剧本信息
// @Summary      获取剧本信息
// @Description  根据剧本ID获取剧本的详细信息
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true  "剧本ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"novel\": {...}}}"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      404       {object}  httputil.ErrorResponse  "剧本不存在"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/detail [get]
func (h *Handler) GetNovel(c *gin.Context) {
	var req GetNovelRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	novelEntity, err := h.novelService.GetNovel(ctx, req.NovelID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "novel not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, httputil.ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	var novelInfo NovelInfo
	novelInfo.FromNovelEntity(novelEntity)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": GetNovelResponseData{
			Novel: novelInfo,
		},
	})
}

// ListNovelsRequest 获取剧本列表请求
type ListNovelsRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`      // 页码，默认1
	PageSize int `form:"page_size" binding:"omitempty,min=1"` // 每页数量，默认10
}

// ListNovelsResponseData 获取剧本列表响应数据
type ListNovelsResponseData struct {
	Novels   []NovelInfo `json:"novels"`    // 剧本列表
	Total    int64       `json:"total"`     // 总数
	Page     int64       `json:"page"`      // 当前页码
	PageSize int64       `json:"page_size"` // 每页数量
}

// ListNovels 获取剧本列表
// @Summary      获取剧本列表
// @Description  根据当前用户获取剧本列表（分页）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        page       query     int     false  "页码，默认1"
// @Param        page_size  query     int     false  "每页数量，默认10"
// @Success      200        {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"novels\": [...], \"total\": 100, \"page\": 1, \"page_size\": 10}}"
// @Failure      400        {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500        {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels [get]
func (h *Handler) ListNovels(c *gin.Context) {
	var req ListNovelsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	// 设置默认值
	page := int64(1)
	pageSize := int64(10)
	if req.Page > 0 {
		page = int64(req.Page)
	}
	if req.PageSize > 0 {
		pageSize = int64(req.PageSize)
	}

	// 从上下文获取用户ID（由认证中间件设置）
	ctx := c.Request.Context()
	userIDStr, exists := ctxutil.GetUserID(ctx)
	if !exists {
		c.JSON(http.StatusUnauthorized, httputil.ErrorResponse{
			Code:    40101,
			Message: "User not authenticated",
		})
		return
	}

	// 调用Service层
	novels, total, err := h.novelService.ListNovelsByUser(ctx, userIDStr, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: "Failed to list novels",
			Detail:  err.Error(),
		})
		return
	}

	// 转换为响应格式
	novelInfos := make([]NovelInfo, len(novels))
	for i, novelEntity := range novels {
		novelInfos[i].FromNovelEntity(novelEntity)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": ListNovelsResponseData{
			Novels:   novelInfos,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// GenerateContentRequest 一键生成内容请求
type GenerateContentRequest struct {
	NovelID        string `json:"novel_id" binding:"required"`              // 小说ID（必填）
	TargetChapters int    `json:"target_chapters" binding:"required,min=1"` // 目标章节数（必填，至少1章）
}

// GenerateContentResponseData 一键生成内容响应数据
type GenerateContentResponseData struct {
	NovelID        string `json:"novel_id"`        // 小说ID
	TargetChapters int    `json:"target_chapters"` // 目标章节数
	Message        string `json:"message"`         // 响应消息
}

// GenerateContent 一键生成内容
// @Summary      一键生成内容
// @Description  自动完成章节切分、人物生成、道具生成、场景和镜头生成。按顺序执行：1. 切分章节 2. 生成人物资源 3. 生成道具资源 4. 生成场景和镜头（为第一个章节）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      GenerateContentRequest  true  "一键生成内容请求"
// @Success      200     {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"内容生成成功\", \"data\": {\"novel_id\": \"...\", \"target_chapters\": 10, \"message\": \"内容生成完成\"}}"
// @Failure      400     {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500     {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/generate-content [post]
func (h *Handler) GenerateContent(c *gin.Context) {
	var req GenerateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	err := h.contentService.GenerateContent(ctx, req.NovelID, req.TargetChapters)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "split chapters failed: failed to find novel":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "no chapters found after splitting":
			code = http.StatusBadRequest
			errorCode = 40003
		}

		c.JSON(code, httputil.ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "内容生成任务已启动",
		"data": GenerateContentResponseData{
			NovelID:        req.NovelID,
			TargetChapters: req.TargetChapters,
			Message:        "内容生成任务已启动，请通过查询接口获取进度",
		},
	})
}

// GetGenerationStatusRequest 获取生成状态请求
type GetGenerationStatusRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
}

// GetGenerationStatusResponseData 获取生成状态响应数据
type GetGenerationStatusResponseData struct {
	Status   string `json:"status"`   // 状态：pending, processing, completed, failed
	Progress int    `json:"progress"` // 进度：0-100
	Message  string `json:"message"`  // 当前步骤说明
}

// GetGenerationStatus 获取生成状态
// @Summary      获取生成状态
// @Description  查询内容生成的当前状态、进度和消息
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/generation-status [get]
func (h *Handler) GetGenerationStatus(c *gin.Context) {
	var req GetGenerationStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	statusInfo, err := h.contentService.GetGenerationStatus(ctx, req.NovelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: "Failed to get generation status",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": GetGenerationStatusResponseData{
			Status:   string(statusInfo.Status),
			Progress: statusInfo.Progress,
			Message:  statusInfo.Message,
		},
	})
}
