package novel

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"lemon/internal/model/novel"
)

// CharacterInfo 角色信息 DTO
type CharacterInfo struct {
	ID         string                      `json:"id"`          // 角色ID
	NovelID    string                      `json:"novel_id"`    // 小说ID
	Name       string                      `json:"name"`        // 角色名称
	Gender     string                      `json:"gender"`      // 性别
	AgeGroup   string                      `json:"age_group"`   // 年龄段
	RoleNumber string                      `json:"role_number"` // 角色编号
	Appearance *novel.CharacterAppearance  `json:"appearance,omitempty"` // 外貌特征
	Clothing   *novel.CharacterClothing    `json:"clothing,omitempty"`   // 服装风格
	CreatedAt  string                      `json:"created_at"`  // 创建时间
	UpdatedAt  string                      `json:"updated_at"`  // 更新时间
}

// toCharacterInfo 将 Character 实体转换为 CharacterInfo DTO
func toCharacterInfo(characterEntity *novel.Character) CharacterInfo {
	return CharacterInfo{
		ID:         characterEntity.ID,
		NovelID:    characterEntity.NovelID,
		Name:       characterEntity.Name,
		Gender:     characterEntity.Gender,
		AgeGroup:   characterEntity.AgeGroup,
		RoleNumber: characterEntity.RoleNumber,
		Appearance: characterEntity.Appearance,
		Clothing:   characterEntity.Clothing,
		CreatedAt:  characterEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  characterEntity.UpdatedAt.Format(time.RFC3339),
	}
}

// toCharacterInfoList 将 Character 实体列表转换为 CharacterInfo DTO 列表
func toCharacterInfoList(characters []*novel.Character) []CharacterInfo {
	list := make([]CharacterInfo, len(characters))
	for i, character := range characters {
		list[i] = toCharacterInfo(character)
	}
	return list
}

// SyncCharactersRequest 同步角色请求
type SyncCharactersRequest struct {
	NovelID     string `json:"novel_id" binding:"required"`     // 小说ID（必填）
	NarrationID string `json:"narration_id" binding:"required"`    // 解说ID（必填）
}

// SyncCharactersResponseData 同步角色响应数据
type SyncCharactersResponseData struct {
	NovelID     string `json:"novel_id"`     // 小说ID
	NarrationID string `json:"narration_id"`  // 解说ID
	Message     string `json:"message"`      // 处理结果消息
}

// SyncCharacters 从章节解说同步角色信息到小说级别
// @Summary      同步角色信息
// @Description  从章节解说同步角色信息到小说级别，提取解说中的角色信息并保存到角色表。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Param        request  body      SyncCharactersRequest  true  "同步角色请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"角色同步成功\", \"data\": {...}}"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/characters/sync [post]
func (h *Handler) SyncCharacters(c *gin.Context) {
	var req SyncCharactersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// 从 URI 获取 novel_id（如果提供）
	novelID := c.Param("novel_id")
	if novelID != "" {
		req.NovelID = novelID
	}

	ctx := c.Request.Context()

	// 调用Service层
	err := h.novelService.SyncCharactersFromNarration(ctx, req.NovelID, req.NarrationID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "find narration":
			code = http.StatusNotFound
			errorCode = 40401
		case err.Error() == "no shots found":
			code = http.StatusBadRequest
			errorCode = 40002
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "角色同步成功",
		"data": SyncCharactersResponseData{
			NovelID:     req.NovelID,
			NarrationID: req.NarrationID,
			Message:     "角色同步成功",
		},
	})
}

// GetCharactersByNovelID 获取小说的所有角色
// @Summary      获取小说角色列表
// @Description  获取小说的所有角色列表。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Param        novel_id  path      string  true  "小说ID"
// @Success      200        {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": [...]}"
// @Failure      400        {object}  ErrorResponse  "请求参数错误"
// @Failure      500        {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/characters [get]
func (h *Handler) GetCharactersByNovelID(c *gin.Context) {
	novelID := c.Param("novel_id")
	if novelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "novel_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	characters, err := h.novelService.GetCharactersByNovelID(ctx, novelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    toCharacterInfoList(characters),
	})
}

// GetCharacterByName 根据名称获取角色
// @Summary      获取角色信息
// @Description  根据名称获取角色信息。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Param        novel_id  path      string  true  "小说ID"
// @Param        name      query     string  true  "角色名称"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {...}}"
// @Failure      400       {object}  ErrorResponse  "请求参数错误"
// @Failure      404       {object}  ErrorResponse  "角色不存在"
// @Failure      500       {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/characters/{name} [get]
func (h *Handler) GetCharacterByName(c *gin.Context) {
	novelID := c.Param("novel_id")
	if novelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "novel_id is required",
		})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40002,
			Message: "name is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	character, err := h.novelService.GetCharacterByName(ctx, novelID, name)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "character not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    toCharacterInfo(character),
	})
}
