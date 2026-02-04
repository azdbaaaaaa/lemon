package resource

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"lemon/internal/service"
)

// DownloadFile 下载文件
// @Summary      下载文件
// @Description  根据资源ID下载文件，返回文件流
// @Tags         资源管理
// @Accept       json
// @Produce      application/octet-stream
// @Param        resource_id  path      string  true  "资源ID"
// @Success      200         {file}    binary  "文件流"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      404         {object}  ErrorResponse  "资源不存在"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/resources/{resource_id}/download [get]
func (h *Handler) DownloadFile(c *gin.Context) {
	resourceID := c.Param("resource_id")
	if resourceID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid resource_id",
		})
		return
	}

	ctx := c.Request.Context()

	// TODO: 从认证中间件中获取用户ID
	// 目前先使用空字符串，视为系统内部请求
	userID := ""

	// 调用Service层
	result, err := h.resourceService.DownloadFile(ctx, &service.DownloadFileRequest{
		UserID:     userID,
		ResourceID: resourceID,
	})
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "资源不存在" {
			code = http.StatusNotFound
			errorCode = 40401
		} else if err.Error() == "无权访问该资源" {
			code = http.StatusForbidden
			errorCode = 40301
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}
	defer result.Data.Close()

	// 设置响应头
	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Disposition", `attachment; filename="`+result.FileName+`"`)
	c.Header("Content-Length", fmt.Sprintf("%d", result.FileSize))

	// 流式传输文件
	_, err = io.Copy(c.Writer, result.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50002,
			Message: "Failed to stream file",
			Detail:  err.Error(),
		})
		return
	}
}
