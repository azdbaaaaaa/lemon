package t2p

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// Config T2P（火山引擎 Text-to-Picture）配置
type Config struct {
	AccessKey      string
	SecretKey      string
	ReqKey         string
	Width          int
	Height         int
	Scale          float64
	DDIMSteps      int
	UsePreLLM      bool
	UseSR          bool
	ReturnURL      bool
	NegativePrompt string
	APIURL         string // API 端点，默认: https://visual.volcengineapi.com
	Region         string // 区域，默认: cn-north-1
}

// ConfigFromEnv 从环境变量创建 T2P 配置
// 支持的环境变量：
//   - VOLCENGINE_ACCESS_KEY: 访问密钥（必需）
//   - VOLCENGINE_SECRET_KEY: 密钥（必需）
//   - T2P_REQ_KEY: 请求密钥（可选，默认: high_aes_general_v21_L）
//   - T2P_WIDTH: 图片宽度（可选，默认: 720）
//   - T2P_HEIGHT: 图片高度（可选，默认: 1280）
//   - T2P_SCALE: 引导尺度（可选，默认: 3.5）
//   - T2P_DDIM_STEPS: 推理步数（可选，默认: 25）
//   - T2P_USE_PRE_LLM: 是否使用预训练LLM优化prompt（可选，默认: false）
//   - T2P_USE_SR: 是否使用超分辨率增强（可选，默认: true）
//   - T2P_RETURN_URL: 返回图片URL（可选，默认: false，返回base64）
//   - T2P_NEGATIVE_PROMPT: 负面提示词（可选）
//   - T2P_API_URL: API 端点（可选，默认: https://visual.volcengineapi.com）
//   - T2P_REGION: 区域（可选，默认: cn-north-1）
func ConfigFromEnv() *Config {
	accessKey := os.Getenv("VOLCENGINE_ACCESS_KEY")
	secretKey := os.Getenv("VOLCENGINE_SECRET_KEY")

	reqKey := os.Getenv("T2P_REQ_KEY")
	if reqKey == "" {
		reqKey = "high_aes_general_v21_L"
	}

	width := 720
	if w := os.Getenv("T2P_WIDTH"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil {
			width = parsed
		}
	}

	height := 1280
	if h := os.Getenv("T2P_HEIGHT"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil {
			height = parsed
		}
	}

	scale := 3.5
	if s := os.Getenv("T2P_SCALE"); s != "" {
		if parsed, err := strconv.ParseFloat(s, 64); err == nil {
			scale = parsed
		}
	}

	ddimSteps := 25
	if d := os.Getenv("T2P_DDIM_STEPS"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			ddimSteps = parsed
		}
	}

	usePreLLM := os.Getenv("T2P_USE_PRE_LLM") == "true"
	useSR := os.Getenv("T2P_USE_SR") != "false" // 默认 true
	returnURL := os.Getenv("T2P_RETURN_URL") == "true"

	negativePrompt := os.Getenv("T2P_NEGATIVE_PROMPT")
	if negativePrompt == "" {
		negativePrompt = "V领, 深V, 锁骨, 脖子, 宫装, 晚礼服, 漏脖子, 低领, 能看见脖子, watermark, (water-marked:1.4), (text:1.5), Signature sketch, (Chinese characters:1.5), (inscription:1.3), letters, (汉字:1.4)，字母，题字，文字，(红色印章:1.4)，logo，对话，标志，对话框，Text, dialog box, watermark, copy, word, letter, subtitle, seal, inscription, English alphabet, nsfw, nude, smooth skin, unblemished skin, mole, low resolution, blurry, worst quality, mutated hands and fingers, poorly drawn face, bad anatomy, distorted hands, limbless, 国旗, national flag."
	}

	apiURL := os.Getenv("T2P_API_URL")
	if apiURL == "" {
		apiURL = "https://visual.volcengineapi.com"
	}

	region := os.Getenv("T2P_REGION")
	if region == "" {
		// 默认区域：根据火山引擎文档，visual 服务通常使用 cn-north-1
		// Python SDK 的 VisualService 默认可能使用 cn-north-1
		region = "cn-north-1"
	}

	return &Config{
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		ReqKey:         reqKey,
		Width:          width,
		Height:         height,
		Scale:          scale,
		DDIMSteps:      ddimSteps,
		UsePreLLM:      usePreLLM,
		UseSR:          useSR,
		ReturnURL:      returnURL,
		NegativePrompt: negativePrompt,
		APIURL:         apiURL,
		Region:         region,
	}
}

// Client T2P（火山引擎 Text-to-Picture）客户端
// 用于调用火山引擎的 visual 服务生成图片
// 参考 Python SDK: volcengine.visual.VisualService
type Client struct {
	config     *Config
	session    *session.Session
	httpClient *http.Client
	apiURL     string
	accessKey  string
	secretKey  string
}

// NewClient 创建 T2P 客户端
// 使用 volcengine-go-sdk 的 session 和 credentials
func NewClient(config *Config) (*Client, error) {
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("VOLCENGINE_ACCESS_KEY and VOLCENGINE_SECRET_KEY are required")
	}

	// 创建 credentials
	creds := credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")

	// 创建 volcengine config
	volcengineConfig := volcengine.NewConfig().
		WithCredentials(creds).
		WithRegion(config.Region)

	// 创建 session
	sess, err := session.NewSession(volcengineConfig)
	if err != nil {
		return nil, fmt.Errorf("create volcengine session: %w", err)
	}

	apiURL := config.APIURL
	if apiURL == "" {
		apiURL = "https://visual.volcengineapi.com"
	}

	return &Client{
		config:     config,
		session:    sess,
		httpClient: &http.Client{Timeout: 300 * time.Second},
		apiURL:     apiURL,
		accessKey:  config.AccessKey,
		secretKey:  config.SecretKey,
	}, nil
}

// GenerateImageRequest 图片生成请求
type GenerateImageRequest struct {
	Prompt         string
	ReqKey         string
	LLMSeed        int
	Seed           int
	Scale          float64
	DDIMSteps      int
	Width          int
	Height         int
	UsePreLLM      bool
	UseSR          bool
	ReturnURL      bool
	NegativePrompt string
	LogoInfo       *LogoInfo
}

// LogoInfo 水印信息
type LogoInfo struct {
	AddLogo         bool    `json:"add_logo"`
	Position        int     `json:"position"`
	Language        int     `json:"language"`
	Opacity         float64 `json:"opacity"`
	LogoTextContent string  `json:"logo_text_content"`
}

// GenerateImageResponse 图片生成响应
type GenerateImageResponse struct {
	ResponseMetadata *ResponseMetadata `json:"ResponseMetadata,omitempty"`
	Data             *ImageData        `json:"data,omitempty"`
}

// ResponseMetadata 响应元数据
type ResponseMetadata struct {
	Error *ErrorInfo `json:"Error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// ImageData 图片数据
type ImageData struct {
	BinaryDataBase64 []string `json:"binary_data_base64,omitempty"`
	ImageURL         []string `json:"image_url,omitempty"`
}

// GenerateImage 生成图片（同步接口）
// 使用火山引擎的 cv_process 接口
// 参考 Python SDK: visual_service.cv_process(form)
func (c *Client) GenerateImage(ctx context.Context, req *GenerateImageRequest) (*GenerateImageResponse, error) {
	// 构建请求参数
	form := map[string]interface{}{
		"req_key":         req.ReqKey,
		"prompt":          req.Prompt,
		"llm_seed":        req.LLMSeed,
		"seed":            req.Seed,
		"scale":           req.Scale,
		"ddim_steps":      req.DDIMSteps,
		"width":           req.Width,
		"height":          req.Height,
		"use_pre_llm":     req.UsePreLLM,
		"use_sr":          req.UseSR,
		"return_url":      req.ReturnURL,
		"negative_prompt": req.NegativePrompt,
	}

	if req.LogoInfo != nil {
		form["logo_info"] = req.LogoInfo
	} else {
		// 默认 logo_info
		form["logo_info"] = map[string]interface{}{
			"add_logo":          false,
			"position":          0,
			"language":          0,
			"opacity":           0.3,
			"logo_text_content": "这里是明水印内容",
		}
	}

	// 序列化请求体
	requestBody, err := json.Marshal(form)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	// 构建 API URL
	apiURL := fmt.Sprintf("%s/?Action=CVProcess&Version=2020-08-26", c.apiURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 实现火山引擎签名
	// 参考: https://www.volcengine.com/docs/6460/6490
	if err := c.signRequest(httpReq, requestBody); err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var apiResp GenerateImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 检查错误
	if apiResp.ResponseMetadata != nil && apiResp.ResponseMetadata.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s",
			apiResp.ResponseMetadata.Error.Code,
			apiResp.ResponseMetadata.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed, status: %d", resp.StatusCode)
	}

	return &apiResp, nil
}

// GenerateImageSimple 简化版本的图片生成（只需要 prompt）
func (c *Client) GenerateImageSimple(ctx context.Context, prompt string) ([]byte, error) {
	req := &GenerateImageRequest{
		Prompt:         prompt,
		ReqKey:         c.config.ReqKey,
		LLMSeed:        -1,
		Seed:           -1,
		Scale:          c.config.Scale,
		DDIMSteps:      c.config.DDIMSteps,
		Width:          c.config.Width,
		Height:         c.config.Height,
		UsePreLLM:      c.config.UsePreLLM,
		UseSR:          c.config.UseSR,
		ReturnURL:      c.config.ReturnURL,
		NegativePrompt: c.config.NegativePrompt,
	}

	resp, err := c.GenerateImage(ctx, req)
	if err != nil {
		return nil, err
	}

	// 提取图片数据
	if resp.Data == nil {
		return nil, fmt.Errorf("no data in response")
	}

	if len(resp.Data.BinaryDataBase64) == 0 {
		return nil, fmt.Errorf("no binary_data_base64 in response")
	}

	// 解码第一张图片
	imageData, err := base64.StdEncoding.DecodeString(resp.Data.BinaryDataBase64[0])
	if err != nil {
		return nil, fmt.Errorf("decode base64 image data: %w", err)
	}

	return imageData, nil
}

// signRequest 为请求添加火山引擎签名
// 参考: https://www.volcengine.com/docs/6460/6490
func (c *Client) signRequest(req *http.Request, body []byte) error {
	// 解析 URL
	u, err := url.Parse(req.URL.String())
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}

	// 获取时间戳
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	date := timestamp[:8]

	// 构建待签名字符串
	// 格式: Method + "\n" + URI + "\n" + QueryString + "\n" + Headers + "\n" + Body
	method := req.Method
	uri := u.Path
	if uri == "" {
		uri = "/"
	}

	// 构建查询字符串（按字典序排序）
	queryParams := u.Query()
	var queryKeys []string
	for k := range queryParams {
		queryKeys = append(queryKeys, k)
	}
	sort.Strings(queryKeys)
	var queryParts []string
	for _, k := range queryKeys {
		values := queryParams[k]
		for _, v := range values {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
		}
	}
	queryString := strings.Join(queryParts, "&")

	// 构建 Headers（按字典序排序）
	headerKeys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		headerKeys = append(headerKeys, strings.ToLower(k))
	}
	sort.Strings(headerKeys)
	var headerParts []string
	for _, k := range headerKeys {
		// 跳过 Host 和 Content-Type（已在其他地方处理）
		if k == "host" || k == "content-type" {
			continue
		}
		values := req.Header[strings.Title(k)]
		for _, v := range values {
			headerParts = append(headerParts, fmt.Sprintf("%s:%s", k, strings.TrimSpace(v)))
		}
	}
	headersString := strings.Join(headerParts, "\n")

	// 构建 Body（请求体）
	bodyString := string(body)

	// 构建待签名字符串
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		method,
		uri,
		queryString,
		headersString,
		bodyString)

	// 计算签名
	// 1. 计算 kDate = HMAC-SHA256(SecretKey, Date)
	kDate := hmacSHA256([]byte(c.secretKey), date)

	// 2. 计算 kRegion = HMAC-SHA256(kDate, Region)
	kRegion := hmacSHA256(kDate, c.config.Region)

	// 3. 计算 kService = HMAC-SHA256(kRegion, "visual")
	kService := hmacSHA256(kRegion, "visual")

	// 4. 计算 kSigning = HMAC-SHA256(kService, "request")
	kSigning := hmacSHA256(kService, "request")

	// 5. 计算 signature = HMAC-SHA256(kSigning, stringToSign)
	signature := hmacSHA256(kSigning, stringToSign)
	signatureHex := fmt.Sprintf("%x", signature)

	// 构建 Authorization header
	// 格式: HMAC-SHA256 Credential={AccessKey}/{Date}/{Region}/{Service}/request, SignedHeaders={SignedHeaders}, Signature={Signature}
	signedHeaders := strings.Join(headerKeys, ";")
	if signedHeaders != "" {
		signedHeaders = ";" + signedHeaders
	}
	authorization := fmt.Sprintf("HMAC-SHA256 Credential=%s/%s/%s/visual/request, SignedHeaders=%s, Signature=%s",
		c.accessKey,
		date,
		c.config.Region,
		signedHeaders,
		signatureHex)

	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-Date", timestamp)

	return nil
}

// hmacSHA256 计算 HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
