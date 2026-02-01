package providers

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/comfyui"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/t2p"
)

// ArkImageProvider Ark 图片生成提供者
// 适配层，调用 ark.ArkImageClient（使用官方 Go SDK）
type ArkImageProvider struct {
	client *ark.ArkImageClient
}

// NewArkImageProvider 创建 Ark 图片生成提供者
// 从环境变量读取配置，创建 ark.ArkImageClient
func NewArkImageProvider() (noveltools.ImageProvider, error) {
	config := ark.ArkImageConfigFromEnv()
	client, err := ark.NewArkImageClient(config)
	if err != nil {
		return nil, fmt.Errorf("create Ark Image client: %w", err)
	}

	return &ArkImageProvider{
		client: client,
	}, nil
}

// GenerateImage 生成图片
// 调用 ark.ArkImageClient.GenerateImageSimple
func (p *ArkImageProvider) GenerateImage(ctx context.Context, prompt, filename string) ([]byte, error) {
	imageData, err := p.client.GenerateImageSimple(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("Ark generate image: %w", err)
	}

	log.Info().
		Str("filename", filename).
		Int("size", len(imageData)).
		Msg("Ark 图片生成成功")

	return imageData, nil
}

// T2PProvider T2P（火山引擎 Text-to-Picture）图片生成提供者
// 适配层，调用 t2p.Client
type T2PProvider struct {
	client *t2p.Client
}

// NewT2PProvider 创建 T2P 提供者
// 从环境变量读取配置，创建 t2p.Client
func NewT2PProvider() (noveltools.ImageProvider, error) {
	config := t2p.ConfigFromEnv()
	client, err := t2p.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create T2P client: %w", err)
	}

	return &T2PProvider{
		client: client,
	}, nil
}

// GenerateImage 生成图片
// 调用 t2p.Client.GenerateImageSimple
func (p *T2PProvider) GenerateImage(ctx context.Context, prompt, filename string) ([]byte, error) {
	imageData, err := p.client.GenerateImageSimple(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("T2P generate image: %w", err)
	}

	log.Info().
		Str("filename", filename).
		Int("size", len(imageData)).
		Msg("T2P 图片生成成功")

	return imageData, nil
}

// ComfyUIProvider ComfyUI 图片生成提供者
// 包装现有的 ComfyUI 客户端
type ComfyUIProvider struct {
	client           *comfyui.Client
	workflowTemplate map[string]interface{}
}

// NewComfyUIProvider 创建 ComfyUI 提供者
func NewComfyUIProvider() (noveltools.ImageProvider, error) {
	config := comfyui.ConfigFromEnv()
	client := comfyui.NewClient(config)

	// 加载工作流模板
	workflowTemplate, err := comfyui.LoadWorkflowJSON(config.WorkflowJSONPath)
	if err != nil {
		return nil, fmt.Errorf("load workflow JSON: %w", err)
	}

	return &ComfyUIProvider{
		client:           client,
		workflowTemplate: workflowTemplate,
	}, nil
}

// GenerateImage 生成图片
func (p *ComfyUIProvider) GenerateImage(ctx context.Context, prompt, filename string) ([]byte, error) {
	// 1. 替换工作流中的正向提示词
	workflow := comfyui.SetPositivePrompt(p.workflowTemplate, prompt)

	// 2. 提交工作流
	result, err := p.client.SubmitWorkflow(ctx, workflow, filename)
	if err != nil {
		return nil, fmt.Errorf("submit workflow: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("submit workflow failed: %s", result.Error)
	}

	// 3. 获取 prompt_id
	promptID, ok := result.Data["prompt_id"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt_id not found in response")
	}

	// 4. 轮询任务状态，等待输出文件名
	outputResult, err := p.client.WaitForOutputFilename(ctx, promptID, filename)
	if err != nil {
		return nil, fmt.Errorf("wait for output filename: %w", err)
	}

	// 5. 下载生成的图片
	imageData, err := p.client.DownloadViewFile(
		ctx,
		outputResult.Filename,
		outputResult.Subfolder,
		outputResult.Type,
		filename,
	)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}

	log.Info().
		Str("filename", filename).
		Int("size", len(imageData)).
		Msg("ComfyUI 图片生成成功")

	return imageData, nil
}
