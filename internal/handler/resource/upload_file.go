package resource

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"lemon/internal/service"
)

// UploadFileResponseData 上传文件响应数据
type UploadFileResponseData struct {
	ResourceID  string `json:"resource_id"`  // 资源ID
	ResourceURL string `json:"resource_url"` // 资源访问URL
	FileSize    int64  `json:"file_size"`    // 文件大小
	FileName    string `json:"file_name"`    // 文件名
}

// UploadFile 上传文件（服务端上传，通过 multipart/form-data）
// @Summary      上传文件
// @Description  通过 multipart/form-data 上传文件到服务端，服务端会保存文件并创建资源记录
// @Tags         资源管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        file      formData  file    true   "上传的文件"
// @Param        user_id   formData  string  false  "用户ID（可选，如果为空则从认证信息中获取）"
// @Success      201       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"文件上传成功\", \"data\": {\"resource_id\": \"...\", \"resource_url\": \"...\", \"file_size\": 1024, \"file_name\": \"...\"}}"
// @Failure      400       {object}  ErrorResponse  "请求参数错误"
// @Failure      500       {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/resources/upload [post]
func (h *Handler) UploadFile(c *gin.Context) {
	// 从 multipart/form-data 中获取文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid file",
			Detail:  err.Error(),
		})
		return
	}

	// 获取用户ID（从表单或认证信息中获取）
	userID := c.PostForm("user_id")
	if userID == "" {
		// TODO: 从认证中间件中获取用户ID
		// 目前先使用默认值或返回错误
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40002,
			Message: "user_id is required",
		})
		return
	}

	// 打开文件
	fileHeader, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40003,
			Message: "Failed to open file",
			Detail:  err.Error(),
		})
		return
	}
	defer fileHeader.Close()

	// 获取文件扩展名
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
	if ext == "" {
		ext = "bin" // 默认扩展名
	}

	// 获取 ContentType
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType([]byte{})
	}

	ctx := c.Request.Context()

	// 调用Service层
	uploadResult, err := h.resourceService.UploadFile(ctx, &service.UploadFileRequest{
		UserID:      userID,
		FileName:    file.Filename,
		ContentType: contentType,
		Ext:         ext,
		Data:        fileHeader,
	})
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if strings.Contains(err.Error(), "文件数据不能为空") {
			code = http.StatusBadRequest
			errorCode = 40004
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "文件上传成功",
		"data": UploadFileResponseData{
			ResourceID:  uploadResult.ResourceID,
			ResourceURL: uploadResult.ResourceURL,
			FileSize:    uploadResult.FileSize,
			FileName:    file.Filename,
		},
	})
}
