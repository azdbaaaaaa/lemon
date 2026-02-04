package resource

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"lemon/internal/service"
)

// ListResourcesRequest 查询资源列表请求
type ListResourcesRequest struct {
	UserID   string `form:"user_id"`   // 用户ID（可选）
	Ext      string `form:"ext"`       // 文件扩展名筛选（可选）
	Status   string `form:"status"`     // 状态筛选（可选）
	Page     int    `form:"page"`      // 页码（默认1）
	PageSize int    `form:"page_size"` // 每页数量（默认20）
}

// ListResourcesResponseData 查询资源列表响应数据
type ListResourcesResponseData struct {
	Resources []ResourceInfo `json:"resources"` // 资源列表
	Total     int64          `json:"total"`    // 总数量
	Page      int            `json:"page"`       // 当前页码
	PageSize  int            `json:"page_size"` // 每页数量
}

// ListResources 查询资源列表
// @Summary      查询资源列表
// @Description  查询资源列表，支持按用户ID、扩展名、状态等条件筛选，支持分页
// @Tags         资源管理
// @Accept       json
// @Produce      json
// @Param        user_id   query     string  false  "用户ID"
// @Param        ext       query     string  false  "文件扩展名筛选"
// @Param        status    query     string  false  "状态筛选"
// @Param        page      query     int     false  "页码（默认1）"
// @Param        page_size query     int     false  "每页数量（默认20，最大100）"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"resources\": [...], \"total\": 100, \"page\": 1, \"page_size\": 20}}"
// @Failure      400       {object}  ErrorResponse  "请求参数错误"
// @Failure      500       {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/resources [get]
func (h *Handler) ListResources(c *gin.Context) {
	var req ListResourcesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid query parameters",
			Detail:  err.Error(),
		})
		return
	}

	// 解析分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 从查询参数中获取分页信息（如果表单绑定失败）
	if req.Page == 0 {
		if pageStr := c.Query("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil {
				req.Page = page
			}
		}
	}
	if req.PageSize == 0 {
		if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
			if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
				req.PageSize = pageSize
			}
		}
	}

	ctx := c.Request.Context()

	// TODO: 从认证中间件中获取用户ID
	// 如果请求中没有指定 user_id，且用户已登录，则使用当前登录用户的ID
	// 目前先使用请求中的 user_id，如果为空则视为系统内部请求
	userID := req.UserID

	// 调用Service层
	result, err := h.resourceService.ListResources(ctx, &service.ListResourcesRequest{
		UserID:   userID,
		Ext:      req.Ext,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": ListResourcesResponseData{
			Resources: toResourceInfoList(result.Resources),
			Total:     result.Total,
			Page:      result.Page,
			PageSize:  result.PageSize,
		},
	})
}
