package comfyui

import (
	"net/url"
	"os"
	"strings"
	"time"
)

// Config ComfyUI 配置
type Config struct {
	APIURL           string        // API URL（如 http://127.0.0.1:8188/api/prompt）
	WorkflowJSONPath string        // 工作流 JSON 模板路径
	Timeout          time.Duration // 请求超时时间
	MaxRetries       int           // 最大重试次数
	RetryDelay       time.Duration // 重试延迟
	PollInterval     time.Duration // 轮询间隔
	MaxWait          time.Duration // 最大等待时间
}

// ConfigFromEnv 从环境变量创建 ComfyUI 配置
// 支持的环境变量：
//   - COMFYUI_API_URL: API URL（可选，默认: http://127.0.0.1:8188/api/prompt）
//   - COMFYUI_WORKFLOW_JSON: 工作流 JSON 模板路径（可选，默认: test/comfyui/image_compact.json）
func ConfigFromEnv() *Config {
	apiURL := os.Getenv("COMFYUI_API_URL")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:8188/api/prompt"
	}

	workflowJSONPath := os.Getenv("COMFYUI_WORKFLOW_JSON")
	if workflowJSONPath == "" {
		workflowJSONPath = "test/comfyui/image_compact.json"
	}

	return &Config{
		APIURL:           normalizePromptURL(apiURL),
		WorkflowJSONPath: workflowJSONPath,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RetryDelay:       1 * time.Second,
		PollInterval:     1 * time.Second,
		MaxWait:          300 * time.Second,
	}
}

// normalizePromptURL 规范化工作流提交端点
// 端点兼容策略：
//   - 支持传入以下形式：
//     1) http://host:port → 归一到 http://host:port/api/prompt
//     2) http://host:port/api → 归一到 http://host:port/api/prompt
//     3) http://host:port/api/prompt → 原样使用
//     4) http://host:port/prompt → 原样使用（部分部署只暴露 /prompt）
//     5) 其他包含 /api/... 的路径 → 回到根并使用 /api/prompt
func normalizePromptURL(urlStr string) string {
	base := strings.TrimSpace(urlStr)
	base = strings.TrimSuffix(base, "/")

	if base == "" {
		base = "http://127.0.0.1:8188"
	}

	// 已是标准 /api/prompt
	if strings.HasSuffix(base, "/api/prompt") || strings.Contains(base, "/api/prompt") {
		return base
	}

	// 明确传入了 /prompt（不带 /api）
	if strings.HasSuffix(base, "/prompt") || (strings.Contains(base, "/prompt") && !strings.Contains(base, "/api")) {
		return base
	}

	// 以 /api 结尾，补齐 /prompt
	if strings.HasSuffix(base, "/api") {
		return base + "/prompt"
	}

	// 包含 /api/... 的其他形式，回到根并统一到 /api/prompt
	if strings.Contains(base, "/api") {
		parts := strings.Split(base, "/api")
		return strings.TrimSuffix(parts[0], "/") + "/api/prompt"
	}

	// 纯主机:端口形式，默认 /api/prompt
	return base + "/api/prompt"
}

// getAPIRoot 返回以 /api 结尾的基础 API 前缀，用于 history/view/upload
func getAPIRoot(promptURL string) string {
	base := strings.TrimSuffix(promptURL, "/")
	if strings.Contains(base, "/api/prompt") {
		parts := strings.Split(base, "/api/prompt")
		return strings.TrimSuffix(parts[0], "/") + "/api"
	}
	if strings.Contains(base, "/prompt") {
		parts := strings.Split(base, "/prompt")
		return strings.TrimSuffix(parts[0], "/") + "/api"
	}
	if strings.Contains(base, "/api") {
		parts := strings.Split(base, "/api")
		return strings.TrimSuffix(parts[0], "/") + "/api"
	}
	return base + "/api"
}

// getFallbackPromptURL 获取备用端点 /prompt
func getFallbackPromptURL(promptURL string) string {
	root := strings.TrimSuffix(promptURL, "/")
	if strings.Contains(root, "/api/prompt") {
		parts := strings.Split(root, "/api/prompt")
		return strings.TrimSuffix(parts[0], "/") + "/prompt"
	}
	if strings.Contains(root, "/prompt") {
		parts := strings.Split(root, "/prompt")
		return strings.TrimSuffix(parts[0], "/") + "/prompt"
	}
	if strings.Contains(root, "/api") {
		parts := strings.Split(root, "/api")
		return strings.TrimSuffix(parts[0], "/") + "/prompt"
	}
	return root + "/prompt"
}

// appendQueryParam 为 URL 追加查询参数
func appendQueryParam(urlStr, key, value string) string {
	if value == "" {
		return urlStr
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		sep := "&"
		if !strings.Contains(urlStr, "?") {
			sep = "?"
		}
		return urlStr + sep + key + "=" + url.QueryEscape(value)
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}
