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

// CharacterInfo 角色信息 DTO
type CharacterInfo struct {
	ID          string                     `json:"id"`                   // 角色ID
	NovelID     string                     `json:"novel_id"`             // 小说ID
	Name        string                     `json:"name"`                 // 角色名称
	Gender      string                     `json:"gender"`               // 性别
	AgeGroup    string                     `json:"age_group"`            // 年龄段
	RoleNumber  string                     `json:"role_number"`          // 角色编号
	Description string                     `json:"description"`          // 角色详细描述
	ImagePrompt string                     `json:"image_prompt"`         // 角色图片提示词
	ImageURL    string                     `json:"image_url,omitempty"`  // 角色图片的直接访问URL
	Appearance  *novel.CharacterAppearance `json:"appearance,omitempty"` // 外貌特征
	Clothing    *novel.CharacterClothing   `json:"clothing,omitempty"`   // 服装风格
	CreatedAt   string                     `json:"created_at"`           // 创建时间
	UpdatedAt   string                     `json:"updated_at"`           // 更新时间
}

// FromCharacterEntity 从 Character 实体创建 CharacterInfo
// imageURL: 角色的图片直接访问URL（如果为空字符串则不填充）
func (info *CharacterInfo) FromCharacterEntity(characterEntity *novel.Character, imageURL string) {
	info.ID = characterEntity.ID
	info.NovelID = characterEntity.NovelID
	info.Name = characterEntity.Name
	info.Gender = characterEntity.Gender
	info.AgeGroup = characterEntity.AgeGroup
	info.RoleNumber = characterEntity.RoleNumber
	info.Description = characterEntity.Description
	info.ImagePrompt = characterEntity.ImagePrompt
	info.Appearance = characterEntity.Appearance
	info.Clothing = characterEntity.Clothing
	info.CreatedAt = characterEntity.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = characterEntity.UpdatedAt.Format(time.RFC3339)
	if imageURL != "" {
		info.ImageURL = imageURL
	}
}

// ToCharacterInfoList 将 Character 实体列表转换为 CharacterInfo 列表
// 会自动查询每个角色的图片信息并获取直接访问URL
func ToCharacterInfoList(ctx context.Context, characters []*novel.Character, novelService interface {
	GetCharacterImages(ctx context.Context, characterID string) ([]*novel.Image, error)
}, resourceService service.ResourceService) []CharacterInfo {
	list := make([]CharacterInfo, len(characters))
	for i, character := range characters {
		// 查询角色的图片（优先使用正视图）
		var imageURL string
		images, err := novelService.GetCharacterImages(ctx, character.ID)
		if err == nil && len(images) > 0 {
			// 优先查找正视图
			var selectedImage *novel.Image
			for _, img := range images {
				if img.CharacterImageSubtype == novel.CharacterImageSubtypeFront {
					selectedImage = img
					break
				}
			}
			// 如果没有正视图，使用第一个图片
			if selectedImage == nil {
				selectedImage = images[0]
			}

			// 获取图片的直接访问URL
			if selectedImage != nil && selectedImage.ImageResourceID != "" {
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
		list[i].FromCharacterEntity(character, imageURL)
	}
	return list
}

// SyncCharactersRequest 同步角色请求
type SyncCharactersRequest struct {
	NovelID   string `json:"novel_id" binding:"required"`   // 小说ID（必填）
	ChapterID string `json:"chapter_id" binding:"required"` // 章节ID（必填）
	Version   int    `json:"version"`                       // 版本号（可选，不传则使用最新版本）
}

// SyncCharactersResponseData 同步角色响应数据
type SyncCharactersResponseData struct {
	NovelID   string `json:"novel_id"`   // 小说ID
	ChapterID string `json:"chapter_id"` // 章节ID
	Version   int    `json:"version"`    // 版本号
	Message   string `json:"message"`    // 处理结果消息
}

// SyncCharacters 从章节解说同步角色信息到小说级别
// @Summary      同步角色信息
// @Description  从章节解说同步角色信息到小说级别，提取解说中的角色信息并保存到角色表。
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      SyncCharactersRequest  true  "同步角色请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"角色同步成功\", \"data\": {...}}"
// @Failure      400      {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500      {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/characters/sync [post]
func (h *Handler) SyncCharacters(c *gin.Context) {
	var req SyncCharactersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// 注意：SyncCharactersFromChapter 方法已删除，因为narration模块已被移除
	// 角色信息现在在生成场景和镜头时自动同步
	// 此接口暂时返回错误，提示用户使用一键生成功能
	c.JSON(http.StatusNotImplemented, httputil.ErrorResponse{
		Code:    50101,
		Message: "此接口已废弃，角色信息在生成场景和镜头时自动同步，请使用一键生成功能",
	})
}

// GetCharactersByNovelIDRequest 获取角色列表请求
type GetCharactersByNovelIDRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
}

// GetCharactersByNovelID 获取小说的所有角色
// @Summary      获取角色列表
// @Description  获取小说的所有角色列表。
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": [...]}"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/characters [get]
func (h *Handler) GetCharactersByNovelID(c *gin.Context) {
	var req GetCharactersByNovelIDRequest
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
	characters, err := h.novelService.GetCharactersByNovelID(ctx, req.NovelID)
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
		"data":    ToCharacterInfoList(ctx, characters, h.novelService, h.resourceService),
	})
}

// GetCharacterByNameRequest 根据名称获取角色请求
type GetCharacterByNameRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
	Name    string `form:"name" binding:"required"`     // 角色名称（必填）
}

// GetCharacterByName 根据名称获取角色
// @Summary      获取角色信息
// @Description  根据名称获取角色信息。
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true  "小说ID"
// @Param        name      query     string  true  "角色名称"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {...}}"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      404       {object}  httputil.ErrorResponse  "角色不存在"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/characters/detail [get]
func (h *Handler) GetCharacterByName(c *gin.Context) {
	var req GetCharacterByNameRequest
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
	character, err := h.novelService.GetCharacterByName(ctx, req.NovelID, req.Name)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "character not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, httputil.ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	var info CharacterInfo
	// 查询角色的图片（优先使用正视图）
	var imageURL string
	images, err := h.novelService.GetCharacterImages(ctx, character.ID)
	if err == nil && len(images) > 0 {
		// 优先查找正视图
		var selectedImage *novel.Image
		for _, img := range images {
			if img.CharacterImageSubtype == novel.CharacterImageSubtypeFront {
				selectedImage = img
				break
			}
		}
		// 如果没有正视图，使用第一个图片
		if selectedImage == nil {
			selectedImage = images[0]
		}

		// 获取图片的直接访问URL
		if selectedImage != nil && selectedImage.ImageResourceID != "" {
			result, err := h.resourceService.GetDownloadURL(ctx, &service.GetDownloadURLRequest{
				UserID:     "", // 系统内部请求
				ResourceID: selectedImage.ImageResourceID,
				ExpiresIn:  24 * time.Hour, // 24小时有效期
			})
			if err == nil && result != nil {
				imageURL = result.DownloadURL
			}
		}
	}
	info.FromCharacterEntity(character, imageURL)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    info,
	})
}

// GenerateCharacterImagesRequest 生成角色图片请求
type GenerateCharacterImagesRequest struct {
	NovelID string `json:"novel_id" binding:"required"` // 小说ID（必填）
}

// GenerateCharacterImages 为小说的所有角色生成图片
// @Summary      生成角色图片
// @Description  为小说的所有角色生成图片（抽卡）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      GenerateCharacterImagesRequest  true  "生成角色图片请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500      {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/characters/images [post]
func (h *Handler) GenerateCharacterImages(c *gin.Context) {
	var req GenerateCharacterImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.novelService.GenerateCharacterImages(ctx, req.NovelID); err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "角色图片生成任务已启动，请通过状态查询接口获取进度",
		"data": gin.H{
			"novel_id": req.NovelID,
		},
	})
}

// GetCharacterImageGenerationStatusRequest 获取角色图片生成状态请求
type GetCharacterImageGenerationStatusRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
}

// CharacterImageGenerationStatus 角色图片生成状态
type CharacterImageGenerationStatus struct {
	CharacterID   string `json:"character_id"`            // 角色ID
	CharacterName string `json:"character_name"`          // 角色名称
	Status        string `json:"status"`                  // 状态：pending, completed, failed
	ErrorMessage  string `json:"error_message,omitempty"` // 错误信息（失败时）
	ImageCount    int    `json:"image_count"`             // 已生成的图片数量
	HasImage      bool   `json:"has_image"`               // 是否有图片
}

// GetCharacterImageGenerationStatus 获取角色图片生成状态
// @Summary      获取角色图片生成状态
// @Description  查询小说的所有角色的图片生成状态
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true   "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/characters/images/status [get]
func (h *Handler) GetCharacterImageGenerationStatus(c *gin.Context) {
	var req GetCharacterImageGenerationStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	characters, err := h.novelService.GetCharactersByNovelID(ctx, req.NovelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	statuses := make([]CharacterImageGenerationStatus, 0, len(characters))
	for _, char := range characters {
		// 查询角色的图片数量
		images, err := h.novelService.GetCharacterImages(ctx, char.ID)
		imageCount := 0
		if err == nil {
			imageCount = len(images)
		}

		statuses = append(statuses, CharacterImageGenerationStatus{
			CharacterID:   char.ID,
			CharacterName: char.Name,
			Status:        string(char.Status),
			ErrorMessage:  char.ErrorMessage,
			ImageCount:    imageCount,
			HasImage:      imageCount > 0,
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
