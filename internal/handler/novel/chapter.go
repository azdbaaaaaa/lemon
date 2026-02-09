package novel

import (
	"net/http"
	"time"

	httputil "lemon/internal/pkg/http"

	"lemon/internal/model/novel"

	"github.com/gin-gonic/gin"
)

// ChapterInfo 章节信息 DTO
type ChapterInfo struct {
	ID                 string `json:"id"`                   // 章节ID
	NovelID            string `json:"novel_id"`             // 小说ID
	UserID             string `json:"user_id"`              // 用户ID
	Sequence           int    `json:"sequence"`             // 章节序号
	Title              string `json:"title"`                // 章节标题
	ChapterText        string `json:"chapter_text"`         // 章节全文
	WordCount          int    `json:"word_count"`           // 章节字数
	ActiveSceneVersion *int   `json:"active_scene_version"` // 当前生效的场景版本号（可选）
	CreatedAt          string `json:"created_at"`           // 创建时间
	UpdatedAt          string `json:"updated_at"`           // 更新时间
}

// FromChapterEntity 从 Chapter 实体创建 ChapterInfo
func (info *ChapterInfo) FromChapterEntity(chapterEntity *novel.Chapter) {
	info.ID = chapterEntity.ID
	info.NovelID = chapterEntity.NovelID
	info.UserID = chapterEntity.UserID
	info.Sequence = chapterEntity.Sequence
	info.Title = chapterEntity.Title
	info.ChapterText = chapterEntity.ChapterText
	info.WordCount = chapterEntity.WordCount
	if chapterEntity.ActiveSceneVersion > 0 {
		info.ActiveSceneVersion = &chapterEntity.ActiveSceneVersion
	}
	info.CreatedAt = chapterEntity.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = chapterEntity.UpdatedAt.Format(time.RFC3339)
}

// ToChapterInfoList 将 Chapter 实体列表转换为 ChapterInfo 列表
func ToChapterInfoList(chapters []*novel.Chapter) []ChapterInfo {
	list := make([]ChapterInfo, len(chapters))
	for i, chapter := range chapters {
		list[i].FromChapterEntity(chapter)
	}
	return list
}

// GetChaptersRequest 获取章节列表请求
type GetChaptersRequest struct {
	NovelID string `form:"novel_id" binding:"required"` // 小说ID（必填）
}

// GetChaptersResponseData 获取章节列表响应数据
type GetChaptersResponseData struct {
	NovelID  string        `json:"novel_id"` // 小说ID
	Chapters []ChapterInfo `json:"chapters"` // 章节列表
	Count    int           `json:"count"`    // 章节数量
}

// GetChapters 获取小说的所有章节
// @Summary      获取章节列表
// @Description  根据小说ID获取该小说的所有章节列表，按序号排序
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        novel_id  query     string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"novel_id\": \"...\", \"chapters\": [...], \"count\": 10}}"
// @Failure      400       {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500       {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/chapters [get]
func (h *Handler) GetChapters(c *gin.Context) {
	var req GetChaptersRequest
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
	chapters, err := h.novelService.GetChapters(ctx, req.NovelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": GetChaptersResponseData{
			NovelID:  req.NovelID,
			Chapters: ToChapterInfoList(chapters),
			Count:    len(chapters),
		},
	})
}

// SplitChaptersRequest 切分章节请求
type SplitChaptersRequest struct {
	NovelID        string `json:"novel_id" binding:"required"`              // 小说ID（必填）
	TargetChapters int    `json:"target_chapters" binding:"required,min=1"` // 目标章节数（必填，至少1章）
}

// SplitChaptersResponseData 切分章节响应数据
type SplitChaptersResponseData struct {
	NovelID        string `json:"novel_id"`        // 小说ID
	TargetChapters int    `json:"target_chapters"` // 目标章节数
	Message        string `json:"message"`         // 响应消息
}

// SplitChapters 根据小说内容切分章节
// @Summary      切分章节
// @Description  根据小说内容切分章节，将小说文本按照目标章节数切分成多个章节。这是小说处理流程的第二步。
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      SplitChaptersRequest  true  "切分章节请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"章节切分成功\", \"data\": {\"novel_id\": \"...\", \"target_chapters\": 10, \"message\": \"已切分为 10 个章节\"}}"
// @Failure      400      {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500      {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/chapters/split [post]
func (h *Handler) SplitChapters(c *gin.Context) {
	var req SplitChaptersRequest
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
	err := h.novelService.SplitChapters(ctx, req.NovelID, req.TargetChapters)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "failed to find novel":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "no chapters split from novel content":
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
		"message": "章节切分成功",
		"data": SplitChaptersResponseData{
			NovelID:        req.NovelID,
			TargetChapters: req.TargetChapters,
			Message:        "已切分为章节",
		},
	})
}
