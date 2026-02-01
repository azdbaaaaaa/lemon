package comfyui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client ComfyUI API 客户端
type Client struct {
	config      *Config
	apiURL      string
	fallbackURL string
	apiRoot     string
	httpClient  *http.Client
}

// NewClient 创建 ComfyUI 客户端
func NewClient(config *Config) *Client {
	apiURL := normalizePromptURL(config.APIURL)
	return &Client{
		config:      config,
		apiURL:      apiURL,
		fallbackURL: getFallbackPromptURL(apiURL),
		apiRoot:     getAPIRoot(apiURL),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SubmitWorkflowResult 工作流提交结果
type SubmitWorkflowResult struct {
	Success bool
	Data    map[string]interface{}
	Error   string
}

// SubmitWorkflow 提交工作流，自动处理端点 405/404 的回退
func (c *Client) SubmitWorkflow(ctx context.Context, workflow map[string]interface{}, filenameParam string) (*SubmitWorkflowResult, error) {
	payload := map[string]interface{}{
		"prompt":    workflow,
		"client_id": "go_client",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow payload: %w", err)
	}

	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.config.RetryDelay)
		}

		// 首选归一端点
		urlPrimary := appendQueryParam(c.apiURL, "image", filenameParam)
		result, err := c.submitRequest(ctx, urlPrimary, payloadBytes)
		if err == nil && result.Success {
			return result, nil
		}

		// 在 404/405 时尝试备用 /prompt
		if result != nil && (result.Error == "404" || result.Error == "405") {
			log.Warn().Str("fallback_url", c.fallbackURL).Msg("提交端点返回错误，尝试回退到备用端点")
			urlFallback := appendQueryParam(c.fallbackURL, "image", filenameParam)
			result2, err2 := c.submitRequest(ctx, urlFallback, payloadBytes)
			if err2 == nil && result2.Success {
				return result2, nil
			}
			if err2 != nil {
				return nil, err2
			}
			if attempt < c.config.MaxRetries-1 {
				continue
			}
			return result2, nil
		}

		if err != nil {
			if attempt < c.config.MaxRetries-1 {
				continue
			}
			return nil, err
		}

		if attempt < c.config.MaxRetries-1 {
			continue
		}
		return result, nil
	}

	return &SubmitWorkflowResult{
		Success: false,
		Error:   "所有重试尝试都失败了",
	}, nil
}

func (c *Client) submitRequest(ctx context.Context, url string, payload []byte) (*SubmitWorkflowResult, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &SubmitWorkflowResult{
			Success: false,
			Error:   fmt.Sprintf("请求错误: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &SubmitWorkflowResult{
			Success: false,
			Error:   fmt.Sprintf("读取响应失败: %v", err),
		}, nil
	}

	if resp.StatusCode == 200 {
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return &SubmitWorkflowResult{
				Success: true,
				Data:    map[string]interface{}{"raw": string(body)},
			}, nil
		}
		return &SubmitWorkflowResult{
			Success: true,
			Data:    data,
		}, nil
	}

	return &SubmitWorkflowResult{
		Success: false,
		Error:   fmt.Sprintf("%d", resp.StatusCode),
	}, nil
}

// WaitForOutputFilenameResult 轮询结果
type WaitForOutputFilenameResult struct {
	Filename  string
	Subfolder string
	Type      string
}

// WaitForOutputFilename 轮询任务状态，等待输出文件名
func (c *Client) WaitForOutputFilename(ctx context.Context, promptID, filenameParam string) (*WaitForOutputFilenameResult, error) {
	url := fmt.Sprintf("%s/history/%s", c.apiRoot, promptID)
	url = appendQueryParam(url, "image", filenameParam)

	endTime := time.Now().Add(c.config.MaxWait)
	for time.Now().Before(endTime) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Warn().Err(err).Msg("轮询历史接口异常")
			time.Sleep(c.config.PollInterval)
			continue
		}

		if resp.StatusCode == 200 {
			var data map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				resp.Body.Close()
				time.Sleep(c.config.PollInterval)
				continue
			}
			resp.Body.Close()

			// 解析输出文件名
			result := c.parseHistoryResponse(data, promptID)
			if result != nil {
				return result, nil
			}
		} else {
			resp.Body.Close()
		}

		time.Sleep(c.config.PollInterval)
	}

	return nil, fmt.Errorf("轮询等待超时，未获取到输出文件名")
}

func (c *Client) parseHistoryResponse(data map[string]interface{}, promptID string) *WaitForOutputFilenameResult {
	var obj map[string]interface{}

	// 尝试从 data[promptID] 获取
	if val, ok := data[promptID].(map[string]interface{}); ok {
		obj = val
	} else if history, ok := data["history"].(map[string]interface{}); ok {
		// 尝试从 data.history[promptID] 获取
		if val, ok := history[promptID].(map[string]interface{}); ok {
			obj = val
		} else {
			// 取第一个值
			for _, v := range history {
				if val, ok := v.(map[string]interface{}); ok {
					obj = val
					break
				}
			}
		}
	}

	if obj == nil {
		return nil
	}

	outputs, ok := obj["outputs"].(map[string]interface{})
	if !ok {
		return nil
	}

	// 遍历所有节点输出
	for _, nodeVal := range outputs {
		node, ok := nodeVal.(map[string]interface{})
		if !ok {
			continue
		}

		images, ok := node["images"].([]interface{})
		if !ok {
			continue
		}

		// 取第一个图片
		for _, img := range images {
			imgMap, ok := img.(map[string]interface{})
			if !ok {
				continue
			}

			filename, _ := imgMap["filename"].(string)
			if filename == "" {
				continue
			}

			subfolder, _ := imgMap["subfolder"].(string)
			type_, _ := imgMap["type"].(string)
			if type_ == "" {
				type_ = "output"
			}

			log.Info().
				Str("filename", filename).
				Str("subfolder", subfolder).
				Str("type", type_).
				Msg("获取到输出文件")

			return &WaitForOutputFilenameResult{
				Filename:  filename,
				Subfolder: subfolder,
				Type:      type_,
			}
		}
	}

	return nil
}

// DownloadViewFile 从 /api/view 下载指定文件
func (c *Client) DownloadViewFile(ctx context.Context, filename, subfolder, type_, filenameParam string) ([]byte, error) {
	params := fmt.Sprintf("filename=%s&type=%s", filename, type_)
	if subfolder != "" {
		params += "&subfolder=" + subfolder
	}
	url := fmt.Sprintf("%s/view?%s", c.apiRoot, params)
	url = appendQueryParam(url, "image", filenameParam)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return data, nil
}

// LoadWorkflowJSON 加载工作流 JSON 模板
func LoadWorkflowJSON(path string) (map[string]interface{}, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("工作流JSON不存在: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow JSON: %w", err)
	}

	var workflow map[string]interface{}
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("unmarshal workflow JSON: %w", err)
	}

	return workflow, nil
}

// SetPositivePrompt 将 workflow 中的正向提示词替换为 promptText
// 优先使用 _meta.title 辨识 Positive 节点，回退到固定节点 ID '12'
func SetPositivePrompt(workflow map[string]interface{}, promptText string) map[string]interface{} {
	// 深拷贝
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		log.Warn().Err(err).Msg("深拷贝工作流失败")
		return workflow
	}

	var wf map[string]interface{}
	if err := json.Unmarshal(workflowBytes, &wf); err != nil {
		log.Warn().Err(err).Msg("反序列化工作流失败")
		return workflow
	}

	// 尝试根据 _meta.title 包含 'Positive Prompt' 的 CLIPTextEncode 节点识别
	var positiveNodeID string
	for nodeID, nodeVal := range wf {
		node, ok := nodeVal.(map[string]interface{})
		if !ok {
			continue
		}

		classType, _ := node["class_type"].(string)
		if classType != "CLIPTextEncode" {
			continue
		}

		meta, _ := node["_meta"].(map[string]interface{})
		title, _ := meta["title"].(string)
		if strings.Contains(title, "Positive") {
			positiveNodeID = nodeID
			break
		}
	}

	// 回退到固定节点 ID '12'
	if positiveNodeID == "" {
		if node12, ok := wf["12"].(map[string]interface{}); ok {
			classType, _ := node12["class_type"].(string)
			if classType == "CLIPTextEncode" {
				positiveNodeID = "12"
			}
		}
	}

	if positiveNodeID == "" {
		log.Warn().Msg("未找到正向提示节点，跳过替换")
		return wf
	}

	node := wf[positiveNodeID].(map[string]interface{})
	inputs, _ := node["inputs"].(map[string]interface{})
	inputs["text"] = promptText

	return wf
}
