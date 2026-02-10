package novel

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"lemon/internal/service"
)

// GenerateShotImagesRequest 生成镜头图片请求
type GenerateShotImagesRequest struct {
	ShotID string `json:"shot_id" binding:"required"` // 镜头ID（必填）
}

// GenerateShotImagesResponseData 生成镜头图片响应数据
type GenerateShotImagesResponseData struct {
	ImageIDs []string `json:"image_ids"` // 生成的图片ID列表
	Count    int      `json:"count"`     // 生成的图片数量
	ShotID   string   `json:"shot_id"`   // 镜头ID
}

// GenerateShotImages 为单个镜头生成首图和尾图
// @Summary      生成镜头图片
// @Description  为单个镜头生成首图和尾图，使用镜头中的 first_image_prompt 和 last_image_prompt
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  path      string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/images [post]
func (h *Handler) GenerateShotImages(c *gin.Context) {
	var req GenerateShotImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid shot_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	imageIDs, err := h.novelService.GenerateShotImages(ctx, req.ShotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "图片生成任务已提交",
		"data": GenerateShotImagesResponseData{
			ImageIDs: imageIDs,
			Count:    len(imageIDs),
			ShotID:   req.ShotID,
		},
	})
}

// GetShotImages 获取镜头的所有图片（首图和尾图）
// @Summary      获取镜头图片
// @Description  获取镜头的所有图片（首图和尾图）
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  query     string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/images [get]
func (h *Handler) GetShotImages(c *gin.Context) {
	shotID := c.Query("shot_id")
	if shotID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "shot_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 获取镜头信息（用于填充场景ID、章节ID、镜头序号等）
	shot, err := h.novelService.GetShotByID(ctx, shotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	// 调用Service层
	images, err := h.novelService.GetShotImages(ctx, shotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	// 转换为 DTO，并获取图片URL
	imageInfos := make([]ImageInfo, 0, len(images))
	for _, img := range images {
		imageInfo := toImageInfo(img)
		// 填充与场景/镜头相关的字段，便于前端按场景查询
		imageInfo.ChapterID = shot.ChapterID
		// 旧字段复用：SceneNumber / ShotNumber 用字符串表示序号
		imageInfo.ShotNumber = fmt.Sprintf("%d", shot.Sequence)

		// 获取图片URL
		if img.ImageResourceID != "" {
			imageURL, _ := h.resourceService.GetDownloadURL(ctx, &service.GetDownloadURLRequest{
				ResourceID: img.ImageResourceID,
			})
			if imageURL != nil {
				imageInfo.ImageURL = imageURL.DownloadURL
			}
		}
		// 不返回 resource_id，只返回 URL
		imageInfo.ImageResourceID = ""
		imageInfos = append(imageInfos, imageInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"shot_id": shotID,
			"images":  imageInfos,
			"count":   len(imageInfos),
		},
	})
}

// GenerateShotAudioRequest 生成镜头音频请求
type GenerateShotAudioRequest struct {
	ShotID string `json:"shot_id" binding:"required"` // 镜头ID（必填）
}

// GenerateShotAudioResponseData 生成镜头音频响应数据
type GenerateShotAudioResponseData struct {
	AudioID string `json:"audio_id"` // 生成的音频ID
	ShotID  string `json:"shot_id"`   // 镜头ID
}

// GenerateShotAudio 为单个镜头生成音频
// @Summary      生成镜头音频
// @Description  为单个镜头生成音频，使用镜头中的 narration 文本
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  path      string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/audio [post]
func (h *Handler) GenerateShotAudio(c *gin.Context) {
	var req GenerateShotAudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid shot_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	audioID, err := h.novelService.GenerateAudioForShot(ctx, req.ShotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "音频生成任务已提交",
		"data": GenerateShotAudioResponseData{
			AudioID: audioID,
			ShotID:  req.ShotID,
		},
	})
}

// GetShotAudios 获取镜头的所有音频
// @Summary      获取镜头音频
// @Description  获取镜头的所有音频
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  query     string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/audios [get]
func (h *Handler) GetShotAudios(c *gin.Context) {
	shotID := c.Query("shot_id")
	if shotID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "shot_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	audios, err := h.novelService.GetAudiosByShot(ctx, shotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	// 转换为 DTO
	audioInfos := make([]AudioInfo, 0, len(audios))
	for _, a := range audios {
		// 获取音频URL
		audioURL, _ := h.resourceService.GetDownloadURL(ctx, &service.GetDownloadURLRequest{
			ResourceID: a.AudioResourceID,
		})
		url := ""
		if audioURL != nil {
			url = audioURL.DownloadURL
		}
		audioInfo := toAudioInfo(a, url)
		// 不返回 resource_id，只返回 URL
		audioInfo.AudioResourceID = ""
		audioInfos = append(audioInfos, audioInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"shot_id": shotID,
			"audios":  audioInfos,
			"count":   len(audioInfos),
		},
	})
}

// GenerateShotVideoRequest 生成镜头视频请求
type GenerateShotVideoRequest struct {
	ShotID string `json:"shot_id" binding:"required"` // 镜头ID（必填）
}

// GenerateShotVideoResponseData 生成镜头视频响应数据
type GenerateShotVideoResponseData struct {
	VideoID string `json:"video_id"` // 生成的视频ID
	ShotID  string `json:"shot_id"`   // 镜头ID
}

// GenerateShotVideo 为单个镜头生成视频
// @Summary      生成镜头视频
// @Description  为单个镜头生成视频，使用镜头的首图或尾图，以及 video_prompt
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  path      string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/video [post]
func (h *Handler) GenerateShotVideo(c *gin.Context) {
	var req GenerateShotVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid shot_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	videoID, err := h.novelService.GenerateVideoForShot(ctx, req.ShotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "视频生成任务已提交",
		"data": GenerateShotVideoResponseData{
			VideoID: videoID,
			ShotID:  req.ShotID,
		},
	})
}

// GenerateShotSubtitleRequest 生成镜头字幕请求
type GenerateShotSubtitleRequest struct {
	ShotID string `json:"shot_id" binding:"required"` // 镜头ID（必填）
}

// GenerateShotSubtitleResponseData 生成镜头字幕响应数据
type GenerateShotSubtitleResponseData struct {
	SubtitleID string `json:"subtitle_id"` // 生成的字幕ID
	ShotID     string `json:"shot_id"`     // 镜头ID
}

// GenerateShotSubtitle 为单个镜头生成字幕
// @Summary      生成镜头字幕
// @Description  为单个镜头生成字幕，需要先有音频记录（包含时间戳数据）
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  body      string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/subtitle [post]
func (h *Handler) GenerateShotSubtitle(c *gin.Context) {
	var req GenerateShotSubtitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid shot_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	subtitleID, err := h.novelService.GenerateSubtitleForShot(ctx, req.ShotID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if strings.Contains(err.Error(), "has no audio") {
			code = http.StatusBadRequest
			errorCode = 40002
		} else if strings.Contains(err.Error(), "has no timestamps") {
			code = http.StatusBadRequest
			errorCode = 40003
		} else if strings.Contains(err.Error(), "find shot") {
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
		"message": "字幕生成成功",
		"data": GenerateShotSubtitleResponseData{
			SubtitleID: subtitleID,
			ShotID:     req.ShotID,
		},
	})
}

// GetShotVideos 获取镜头的所有视频
// @Summary      获取镜头视频
// @Description  获取镜头的所有视频
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  query     string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots/videos [get]
func (h *Handler) GetShotVideos(c *gin.Context) {
	shotID := c.Query("shot_id")
	if shotID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "shot_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	videos, err := h.novelService.GetVideosByShot(ctx, shotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	// 转换为 DTO
	videoInfos := toVideoInfoList(ctx, videos, h.resourceService)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"shot_id": shotID,
			"videos":  videoInfos,
			"count":   len(videoInfos),
		},
	})
}

// GetShot 获取镜头详情
// @Summary      获取镜头详情
// @Description  获取镜头的详细信息
// @Tags         镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  query     string  true  "镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots [get]
func (h *Handler) GetShot(c *gin.Context) {
	shotID := c.Query("shot_id")
	if shotID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "shot_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层（通过 scene service 获取）
	shot, err := h.novelService.GetShotByID(ctx, shotID)
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
		"data": gin.H{
			"shot": shot,
		},
	})
}
