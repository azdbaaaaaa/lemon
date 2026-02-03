package ark

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
)

// ArkVideoConfig Ark 视频生成配置
type ArkVideoConfig struct {
	APIKey  string // API Key（必需）
	BaseURL string // API 基础 URL（可选，默认: https://ark.cn-beijing.volces.com/api/v3）
	Model   string // 模型名称（可选，默认: doubao-seedance-1-0-lite-i2v-250428）
}

// ArkVideoConfigFromEnv 从环境变量创建 Ark 视频生成配置
// 支持的环境变量：
//   - ARK_API_KEY: API Key（必需，用于视频生成）
//   - ARK_VIDEO_MODEL: 视频生成模型名称（可选，默认: doubao-seedance-1-0-lite-i2v-250428）
//   - ARK_BASE_URL: API 基础 URL（可选，默认: https://ark.cn-beijing.volces.com/api/v3）
func ArkVideoConfigFromEnv() *ArkVideoConfig {
	apiKey := os.Getenv("ARK_API_KEY")
	model := os.Getenv("ARK_VIDEO_MODEL")
	baseURL := os.Getenv("ARK_BASE_URL")

	if model == "" {
		model = "doubao-seedance-1-0-lite-i2v-250428" // 默认视频生成模型
	}
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	return &ArkVideoConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}
}

// ArkVideoClient Ark 视频生成客户端
// 用于调用火山引擎的 Ark API 生成视频（image-to-video）
// 参考 Python SDK: volcenginesdkarkruntime.Ark().content_generation.tasks.create()
type ArkVideoClient struct {
	client  *arkruntime.Client
	model   string
	baseURL string
	apiKey  string
}

// NewArkVideoClient 创建 Ark 视频生成客户端
func NewArkVideoClient(config *ArkVideoConfig) (*ArkVideoClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("ARK_API_KEY is required")
	}

	// 创建客户端选项
	var opts []arkruntime.ConfigOption
	if config.BaseURL != "" {
		opts = append(opts, arkruntime.WithBaseUrl(config.BaseURL))
	}

	// 使用 API Key 创建客户端
	arkClient := arkruntime.NewClientWithApiKey(config.APIKey, opts...)

	return &ArkVideoClient{
		client:  arkClient,
		model:   config.Model,
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
	}, nil
}

// GenerateVideoFromImage 从单张图片生成视频（同步等待）
// 对应 Python: client.content_generation.tasks.create() + 轮询等待
//
// 实现流程：
// 1. 调用 tasks.create() 提交任务（异步 API，返回 task_id）
// 2. 在函数内部轮询 tasks.get(task_id) 直到任务完成
// 3. 下载视频数据并返回
//
// Args:
//   - ctx: 上下文
//   - imageDataURL: 图片的 data URL（base64 编码，格式: data:image/jpeg;base64,...）
//   - duration: 视频时长（秒，最大 12 秒）
//   - prompt: 视频生成提示词（可选）
//
// Returns:
//   - []byte: 视频数据
//   - error: 错误信息
func (c *ArkVideoClient) GenerateVideoFromImage(ctx context.Context, imageDataURL string, duration int, prompt string) ([]byte, error) {
	// 限制 duration 最大为 12 秒
	limitedDuration := duration
	if limitedDuration > 12 {
		limitedDuration = 12
		log.Warn().Int("original", duration).Int("limited", limitedDuration).Msg("视频时长超过限制，已调整为 12 秒")
	}

	// 构建提示词
	// 如果 prompt 为空，使用更详细的默认 prompt，包含镜头运动、转场效果、动作描述
	if prompt == "" {
		prompt = "画面有明显的动态效果，镜头缓慢推进，人物有自然的动作和表情变化，背景有轻微的运动感，整体画面流畅自然，动作幅度适中"
	}

	// 1. 提交任务（异步 API，只返回 task_id）
	// 注意：Go SDK 可能没有 content_generation.tasks 的 API，需要直接使用 HTTP 请求
	// 这里先尝试使用 SDK，如果不行再改用 HTTP 请求
	taskID, err := c.createVideoTask(ctx, imageDataURL, prompt, limitedDuration, "9:16")
	if err != nil {
		return nil, fmt.Errorf("failed to create video task: %w", err)
	}

	log.Info().Str("task_id", taskID).Msg("视频生成任务提交成功")

	// 2. 同步轮询等待任务完成（在函数内部，阻塞等待）
	maxWaitTime := 30 * time.Minute // 最大等待 10 分钟（视频生成可能需要较长时间）
	pollInterval := 5 * time.Second // 每 5 秒轮询一次
	startTime := time.Now()

	for {
		// 检查超时
		if time.Since(startTime) > maxWaitTime {
			return nil, fmt.Errorf("video generation timeout after %v", maxWaitTime)
		}

		// 查询任务状态
		status, videoURL, err := c.getTaskStatus(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task status: %w", err)
		}

		if status == "succeeded" || status == "completed" {
			// 3. 下载视频数据
			if videoURL == "" {
				return nil, fmt.Errorf("video URL is empty")
			}
			videoData, err := c.downloadVideo(ctx, videoURL)
			if err != nil {
				return nil, fmt.Errorf("failed to download video: %w", err)
			}
			log.Info().Str("task_id", taskID).Int("size", len(videoData)).Msg("视频生成成功并下载完成")
			return videoData, nil
		} else if status == "failed" {
			return nil, fmt.Errorf("video generation task failed: task_id=%s", taskID)
		}

		// 等待一段时间后继续轮询
		log.Debug().Str("task_id", taskID).Str("status", status).Msg("视频生成中，继续等待...")
		time.Sleep(pollInterval)
	}
}

// createVideoTask 创建视频生成任务
// 使用 HTTP 请求直接调用 Ark API（因为 Go SDK 可能没有 content_generation.tasks 的 API）
// 参考官方文档: https://www.volcengine.com/docs/82379/1520757
func (c *ArkVideoClient) createVideoTask(ctx context.Context, imageDataURL string, prompt string, duration int, ratio string) (string, error) {
	// 构建请求体
	// 参考官方文档 curl 示例
	requestBody := map[string]interface{}{
		"model": c.model,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": prompt,
			},
			{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": imageDataURL,
				},
			},
		},
		"ratio":     ratio,    // 视频比例，如 "9:16" 或 "adaptive"
		"duration":  duration, // 视频时长（秒）
		"watermark": false,    // 是否添加水印
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	// 构建 API URL
	// 参考官方文档: https://www.volcengine.com/docs/82379/1520757
	// 创建视频生成任务 API 路径: POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks
	// 确保 baseURL 格式正确（移除末尾的斜杠）
	baseURL := strings.TrimSuffix(c.baseURL, "/")
	apiURL := fmt.Sprintf("%s/contents/generations/tasks", baseURL)

	// 构建日志友好的请求体（隐藏 base64 图片数据）
	logRequestBody := map[string]interface{}{
		"model": c.model,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": prompt,
			},
			{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": "[base64 image data...]",
				},
			},
		},
		"ratio":     ratio,
		"duration":  duration,
		"watermark": false,
	}
	logBodyData, _ := json.Marshal(logRequestBody)

	log.Debug().
		Str("api_url", apiURL).
		Str("model", c.model).
		Str("request_body", string(logBodyData)).
		Msg("创建视频生成任务")

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	// 创建视频任务可能需要较长时间，增加超时时间到 10 分钟
	// 视频生成任务创建时，服务器需要处理图片数据（base64 编码），可能需要较长时间
	// 特别是当图片较大时，服务器处理时间会更长
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("url", apiURL).
			Str("response_body", string(body)).
			Msg("API 请求失败")
		return "", fmt.Errorf("API request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if apiResp.ID == "" {
		return "", fmt.Errorf("task ID is empty in response")
	}

	return apiResp.ID, nil
}

// getTaskStatus 查询任务状态
func (c *ArkVideoClient) getTaskStatus(ctx context.Context, taskID string) (status string, videoURL string, err error) {
	// 构建 API URL
	// 参考官方文档: https://www.volcengine.com/docs/82379/1521309
	// 查询视频生成任务 API 路径: GET /api/v3/contents/generations/tasks/{task_id}
	// 确保 baseURL 格式正确（移除末尾的斜杠）
	baseURL := strings.TrimSuffix(c.baseURL, "/")
	apiURL := fmt.Sprintf("%s/contents/generations/tasks/%s", baseURL, taskID)

	log.Debug().
		Str("api_url", apiURL).
		Str("task_id", taskID).
		Msg("查询视频生成任务状态")

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("url", apiURL).
			Str("task_id", taskID).
			Str("response_body", string(body)).
			Msg("查询任务状态失败")
		return "", "", fmt.Errorf("API request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp struct {
		Status  string `json:"status"`
		Content struct {
			VideoURL string `json:"video_url"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	return apiResp.Status, apiResp.Content.VideoURL, nil
}

// downloadVideo 下载视频
func (c *ArkVideoClient) downloadVideo(ctx context.Context, videoURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download video: status code %d", resp.StatusCode)
	}

	videoData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read video data: %w", err)
	}

	return videoData, nil
}

// ConvertImageToDataURL 将图片数据转换为 data URL
func ConvertImageToDataURL(imageData []byte, mimeType string) string {
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
}
