package novel

import (
	"context"
	"net/http"
	"time"

	httputil "lemon/internal/pkg/http"

	"lemon/internal/model/novel"
	"lemon/internal/service"

	"github.com/gin-gonic/gin"
)

// PropInfo 道具信息 DTO
type PropInfo struct {
	ID          string `json:"id"`                  // 道具ID
	NovelID     string `json:"novel_id"`            // 小说ID
	Name        string `json:"name"`                // 道具名称
	Category    string `json:"category,omitempty"`  // 道具类别
	Description string `json:"description"`         // 道具详细描述
	ImagePrompt string `json:"image_prompt"`        // 道具图片提示词
	ImageURL    string `json:"image_url,omitempty"` // 道具图片的直接访问URL
	CreatedAt   string `json:"created_at"`          // 创建时间
	UpdatedAt   string `json:"updated_at"`          // 更新时间
}

// FromPropEntity 从 Prop 实体创建 PropInfo
// imageURL: 道具的图片直接访问URL（如果为空字符串则不填充）
func (info *PropInfo) FromPropEntity(propEntity *novel.Prop, imageURL string) {
	info.ID = propEntity.ID
	info.NovelID = propEntity.NovelID
	info.Name = propEntity.Name
	info.Category = propEntity.Category
	info.Description = propEntity.Description
	info.ImagePrompt = propEntity.ImagePrompt
	info.CreatedAt = propEntity.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = propEntity.UpdatedAt.Format(time.RFC3339)
	if imageURL != "" {
		info.ImageURL = imageURL
	}
}

// ToPropInfoList 将 Prop 实体列表转换为 PropInfo 列表
// 会自动查询每个道具的图片信息并获取直接访问URL
func ToPropInfoList(ctx context.Context, props []*novel.Prop, novelService interface {
	GetPropImages(ctx context.Context, propID string) ([]*novel.Image, error)
}, resourceService service.ResourceService) []PropInfo {
	list := make([]PropInfo, len(props))
	for i, prop := range props {
		// 查询道具的图片
		var imageURL string
		images, err := novelService.GetPropImages(ctx, prop.ID)
		if err == nil && len(images) > 0 {
			// 使用第一个图片
			selectedImage := images[0]
			if selectedImage.ImageResourceID != "" {
				// 获取图片的直接访问URL
				result, err := resourceService.GetDownloadURL(ctx, &service.GetDownloadURLRequest{
					UserID:     "", // 系统内部请求
					ResourceID: selectedImage.ImageResourceID,
					ExpiresIn:  24 * time.Hour, // 24小时有效期
				})
				if err == nil && result != nil {
					imageURL = result.DownloadURL
				}
			}
		}
		list[i].FromPropEntity(prop, imageURL)
	}
	return list
}

// GetPropsByNovelIDRequest 获取道具列表请求
type GetPropsByNovelIDRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
}

// GetPropsByNovelID 获取小说的所有道具
// @Summary      获取道具列表
// @Description  获取小说的所有道具列表
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/props [get]
func (h *Handler) GetPropsByNovelID(c *gin.Context) {
	var req GetPropsByNovelIDRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	props, err := h.novelService.GetPropsByNovelID(ctx, req.NovelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    ToPropInfoList(ctx, props, h.novelService, h.resourceService),
	})
}

// GeneratePropImagesRequest 生成道具图片请求
type GeneratePropImagesRequest struct {
	NovelID string `json:"novel_id" binding:"required"` // 小说ID（必填）
}

// GeneratePropImages 为小说的所有道具生成图片
// @Summary      生成道具图片
// @Description  为小说的所有道具生成图片（抽卡）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      GeneratePropImagesRequest  true  "生成道具图片请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500      {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/props/images [post]
func (h *Handler) GeneratePropImages(c *gin.Context) {
	var req GeneratePropImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.novelService.GeneratePropImages(ctx, req.NovelID); err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "道具图片生成任务已启动，请通过状态查询接口获取进度",
		"data": gin.H{
			"novel_id": req.NovelID,
		},
	})
}

// GetPropImageGenerationStatusRequest 获取道具图片生成状态请求
type GetPropImageGenerationStatusRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
}

// PropImageGenerationStatus 道具图片生成状态
type PropImageGenerationStatus struct {
	PropID       string `json:"prop_id"`                 // 道具ID
	PropName     string `json:"prop_name"`               // 道具名称
	Status       string `json:"status"`                  // 状态：pending, completed, failed
	ErrorMessage string `json:"error_message,omitempty"` // 错误信息（失败时）
}

// GetPropImageGenerationStatus 获取道具图片生成状态
// @Summary      获取道具图片生成状态
// @Description  查询小说的所有道具的图片生成状态
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true   "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/props/images/status [get]
func (h *Handler) GetPropImageGenerationStatus(c *gin.Context) {
	var req GetPropImageGenerationStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	// 通过 GetPropsByNovelID 获取道具列表（如果不存在，需要添加该方法）
	// 或者直接使用已有的 GetProps handler 方法
	// 暂时使用 GetPropsByNovelID，如果不存在需要添加
	props, err := h.novelService.GetPropsByNovelID(ctx, req.NovelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	statuses := make([]PropImageGenerationStatus, 0, len(props))
	for _, prop := range props {
		statuses = append(statuses, PropImageGenerationStatus{
			PropID:       prop.ID,
			PropName:     prop.Name,
			Status:       string(prop.Status),
			ErrorMessage: prop.ErrorMessage,
		})
	}

	// 统计状态
	total := len(statuses)
	pending := 0
	completed := 0
	failed := 0
	for _, status := range statuses {
		switch status.Status {
		case "pending":
			pending++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"novel_id": req.NovelID,
			"statuses": statuses,
			"summary": gin.H{
				"total":     total,
				"pending":   pending,
				"completed": completed,
				"failed":    failed,
			},
		},
	})
}
