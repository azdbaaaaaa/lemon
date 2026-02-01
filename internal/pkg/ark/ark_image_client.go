package ark

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// ArkImageConfig Ark 图片生成配置
type ArkImageConfig struct {
	APIKey  string // API Key（必需）
	BaseURL string // API 基础 URL（可选，默认: https://ark.cn-beijing.volces.com/api/v3）
	Model   string // 模型名称（可选，默认: doubao-seedream-3-0-t2i-250415）
}

// ArkImageConfigFromEnv 从环境变量创建 Ark 图片生成配置
// 支持的环境变量：
//   - ARK_API_KEY: API Key（必需，用于图片生成）
//   - ARK_IMAGE_MODEL: 图片生成模型名称（可选，默认: doubao-seedream-3-0-t2i-250415）
//   - ARK_BASE_URL: API 基础 URL（可选，默认: https://ark.cn-beijing.volces.com/api/v3）
func ArkImageConfigFromEnv() *ArkImageConfig {
	apiKey := os.Getenv("ARK_API_KEY")
	model := os.Getenv("ARK_IMAGE_MODEL")
	baseURL := os.Getenv("ARK_BASE_URL")

	if model == "" {
		model = "doubao-seedream-3-0-t2i-250415" // 默认图片生成模型
	}
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	return &ArkImageConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}
}

// ArkImageClient Ark 图片生成客户端
// 用于调用火山引擎的 Ark API 生成图片
// 参考 Python SDK: volcenginesdkarkruntime.Ark().images.generate()
type ArkImageClient struct {
	client *arkruntime.Client
	model  string
}

// NewArkImageClient 创建 Ark 图片生成客户端
func NewArkImageClient(config *ArkImageConfig) (*ArkImageClient, error) {
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

	return &ArkImageClient{
		client: arkClient,
		model:  config.Model,
	}, nil
}

// GenerateImage 生成图片（同步接口）
// 对应 Python SDK: client.images.generate()
func (c *ArkImageClient) GenerateImage(ctx context.Context, prompt string, size string, watermark bool) ([]byte, error) {
	// 设置默认值
	if size == "" {
		size = "720x1280"
	}

	responseFormat := "b64_json"

	// 构建请求参数（使用 Go SDK 的实际类型）
	input := model.GenerateImagesRequest{
		Model:          c.model,
		Prompt:         prompt,
		Size:           &size,
		ResponseFormat: &responseFormat,
		Watermark:      &watermark,
	}

	// 调用 API（使用 Go SDK 的实际方法名）
	output, err := c.client.GenerateImages(ctx, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to call Ark GenerateImages API")
		return nil, fmt.Errorf("Ark GenerateImages API call failed: %w", err)
	}

	// 提取图片数据
	if len(output.Data) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}

	// 获取第一张图片的 base64 数据
	firstImage := output.Data[0]
	if firstImage.B64Json == nil {
		return nil, fmt.Errorf("no b64_json in response data")
	}

	// 解码 base64 图片数据
	imageData, err := base64.StdEncoding.DecodeString(*firstImage.B64Json)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	return imageData, nil
}

// GenerateImageSimple 简化版本的图片生成（只需要 prompt）
func (c *ArkImageClient) GenerateImageSimple(ctx context.Context, prompt string) ([]byte, error) {
	return c.GenerateImage(ctx, prompt, "720x1280", false)
}
