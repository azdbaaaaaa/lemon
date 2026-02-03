package tts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/pkg/id"
)

// Config TTS 配置
type Config struct {
	APIURL      string // API 地址，默认: https://openspeech.bytedance.com/api/v1/tts
	AccessToken string // 访问令牌（必需）
	AppID       string // 应用ID（可选）
	Cluster     string // 集群名称，默认: volcano_tts
	VoiceType   string // 语音类型，默认: BV115_streaming
	SampleRate  int    // 采样率，默认: 44100
}

// ConfigFromEnv 从环境变量创建 TTSConfig
// 支持的环境变量：
//   - TTS_ACCESS_TOKEN: 访问令牌（必需）
//   - TTS_APP_ID: 应用ID（可选）
//   - TTS_VOICE_TYPE: 语音类型（可选，默认: BV115_streaming）
//   - TTS_CLUSTER: 集群名称（可选，默认: volcano_tts）
//   - TTS_SAMPLE_RATE: 采样率（可选，默认: 44100）
//   - TTS_API_URL: API 地址（可选，默认: https://openspeech.bytedance.com/api/v1/tts）
func ConfigFromEnv() Config {
	accessToken := os.Getenv("TTS_ACCESS_TOKEN")
	appID := os.Getenv("TTS_APP_ID")
	voiceType := os.Getenv("TTS_VOICE_TYPE")
	cluster := os.Getenv("TTS_CLUSTER")
	apiURL := os.Getenv("TTS_API_URL")
	sampleRateStr := os.Getenv("TTS_SAMPLE_RATE")

	if voiceType == "" {
		voiceType = "BV115_streaming"
	}
	if cluster == "" {
		cluster = "volcano_tts"
	}
	if apiURL == "" {
		apiURL = "https://openspeech.bytedance.com/api/v1/tts"
	}

	sampleRate := 44100
	if sampleRateStr != "" {
		if parsed, err := strconv.Atoi(sampleRateStr); err == nil {
			sampleRate = parsed
		}
	}

	return Config{
		APIURL:      apiURL,
		AccessToken: accessToken,
		AppID:       appID,
		Cluster:     cluster,
		VoiceType:   voiceType,
		SampleRate:  sampleRate,
	}
}

// Client TTS 客户端封装
// 用于调用火山引擎的 TTS API（文本转语音）
// 参考: https://openspeech.bytedance.com/api/v1/tts
type Client struct {
	apiURL      string
	accessToken string
	appID       string
	cluster     string
	voiceType   string
	sampleRate  int
	httpClient  *http.Client
}

// NewClient 创建 TTS 客户端
func NewClient(config Config) (*Client, error) {
	if config.AccessToken == "" {
		return nil, fmt.Errorf("TTS access token is required")
	}

	apiURL := config.APIURL
	if apiURL == "" {
		apiURL = "https://openspeech.bytedance.com/api/v1/tts"
	}

	cluster := config.Cluster
	if cluster == "" {
		cluster = "volcano_tts"
	}

	voiceType := config.VoiceType
	if voiceType == "" {
		voiceType = "BV115_streaming"
	}

	sampleRate := config.SampleRate
	if sampleRate == 0 {
		sampleRate = 44100
	}

	return &Client{
		apiURL:      apiURL,
		accessToken: config.AccessToken,
		appID:       config.AppID,
		cluster:     cluster,
		voiceType:   voiceType,
		sampleRate:  sampleRate,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Result TTS生成结果
type Result struct {
	Success       bool           `json:"success"`        // 是否成功
	AudioData     []byte         `json:"-"`              // 音频数据（二进制，不序列化到 JSON）
	Duration      float64        `json:"duration"`       // 音频时长（秒）
	TimestampData *TimestampData `json:"timestamp_data"` // 时间戳数据
	ErrorMessage  string         `json:"error_message"`  // 错误信息
}

// TimestampData 时间戳数据
type TimestampData struct {
	Text                string          `json:"text"`                 // 原始文本
	Duration            float64         `json:"duration"`             // 音频时长（秒）
	CharacterTimestamps []CharTimestamp `json:"character_timestamps"` // 字符级时间戳
	GeneratedAt         time.Time       `json:"generated_at"`         // 生成时间
}

// CharTimestamp 字符时间戳
type CharTimestamp struct {
	Character string  `json:"character"`  // 字符
	StartTime float64 `json:"start_time"` // 开始时间（秒）
	EndTime   float64 `json:"end_time"`   // 结束时间（秒）
}

// GenerateVoiceWithTimestamps 生成语音并获取时间戳
// 返回音频数据和时长，不保存到文件
func (c *Client) GenerateVoiceWithTimestamps(
	ctx context.Context,
	text string,
	speedRatio float64,
) (*Result, error) {
	result := &Result{
		Success: false,
	}

	// 1. 构建请求配置
	requestID := id.New()
	requestConfig := c.buildRequestConfig(text, requestID, speedRatio)

	// 2. 发送 HTTP 请求
	reqBody, err := json.Marshal(requestConfig)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to marshal request: %v", err)
		return result, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, io.NopCloser(
		&requestBodyReader{data: reqBody}))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		return result, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer; %s", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	log.Debug().
		Str("request_id", requestID).
		Str("text", text).
		Msg("sending TTS request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to send request: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	// 3. 解析响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to read response: %v", err)
		return result, err
	}

	if resp.StatusCode != http.StatusOK {
		result.ErrorMessage = fmt.Sprintf("API request failed, status: %d, body: %s", resp.StatusCode, string(respBody))
		return result, fmt.Errorf("API request failed: status %d", resp.StatusCode)
	}

	// 4. 解析 JSON 响应
	var apiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// 尝试修复 JSON（参考 Python 代码的修复逻辑）
		fixedBody := c.fixJSON(string(respBody))
		if err := json.Unmarshal([]byte(fixedBody), &apiResp); err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to parse JSON response: %v", err)
			return result, err
		}
	}

	// 5. 检查响应状态
	code, _ := apiResp["code"].(float64)
	if code != 3000 {
		message, _ := apiResp["message"].(string)
		if message == "" {
			message = "unknown error"
		}
		result.ErrorMessage = fmt.Sprintf("API response error: %s (code: %.0f)", message, code)
		return result, fmt.Errorf("API response error: %s", message)
	}

	// 6. 提取音频数据
	audioDataBase64, ok := apiResp["data"].(string)
	if !ok {
		result.ErrorMessage = "audio data not found in response"
		return result, fmt.Errorf("audio data not found")
	}

	audioData, err := base64.StdEncoding.DecodeString(audioDataBase64)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to decode audio data: %v", err)
		return result, err
	}

	// 7. 解析时间戳数据和音频时长
	timestampData, duration := c.parseTimestampData(apiResp, text)

	result.Success = true
	result.AudioData = audioData
	result.Duration = duration
	result.TimestampData = timestampData

	return result, nil
}

// buildRequestConfig 构建请求配置
// 参考官方文档: https://openspeech.bytedance.com/api/v1/tts
func (c *Client) buildRequestConfig(text, requestID string, speedRatio float64) map[string]interface{} {
	appConfig := map[string]interface{}{
		"token":   c.accessToken,
		"cluster": c.cluster,
	}
	if c.appID != "" {
		appConfig["appid"] = c.appID
	}

	// 根据官方文档格式构建请求
	audioConfig := map[string]interface{}{
		"voice_type":       c.voiceType,
		"encoding":         "mp3",
		"compression_rate": 1,
		"rate":             c.sampleRate,
		"speed_ratio":      speedRatio,
		"volume_ratio":     1.0,
		"pitch_ratio":      1.0,
		"language":         "cn",
	}

	requestConfig := map[string]interface{}{
		"reqid":            requestID,
		"text":             text,
		"text_type":        "plain",
		"operation":        "query",
		"silence_duration": "125",
		"with_frontend":    "1",
		"frontend_type":    "unitTson",
		"pure_english_opt": "1",
	}

	// extra_param 是可选的，如果需要可以添加
	// extraParam := map[string]interface{}{
	// 	"disable_emoji_filter": true,
	// }
	// extraParamJSON, _ := json.Marshal(extraParam)
	// requestConfig["extra_param"] = string(extraParamJSON)

	return map[string]interface{}{
		"app":     appConfig,
		"user":    map[string]interface{}{"uid": requestID},
		"audio":   audioConfig,
		"request": requestConfig,
	}
}

// parseTimestampData 解析时间戳数据和音频时长
// 返回时间戳数据和音频时长（秒）
func (c *Client) parseTimestampData(apiResp map[string]interface{}, text string) (*TimestampData, float64) {
	timestampData := &TimestampData{
		Text:                text,
		Duration:            0,
		CharacterTimestamps: []CharTimestamp{},
		GeneratedAt:         time.Now(),
	}

	var duration float64

	// 从 addition 字段获取时间戳信息
	addition, ok := apiResp["addition"].(map[string]interface{})
	if !ok {
		return timestampData, duration
	}

	// 获取 duration（单位：毫秒，需要转换为秒）
	// duration 可能是字符串或数字
	if durationStr, ok := addition["duration"].(string); ok {
		if parsed, err := strconv.ParseFloat(durationStr, 64); err == nil {
			duration = parsed / 1000.0 // 转换为秒
			timestampData.Duration = duration
		}
	} else if durationNum, ok := addition["duration"].(float64); ok {
		duration = durationNum / 1000.0 // 转换为秒
		timestampData.Duration = duration
	}

	// 解析 frontend 字段（包含字符级时间戳）
	frontendStr, ok := addition["frontend"].(string)
	if !ok {
		// 如果不是字符串，可能是已经解析的对象
		if frontendObj, ok := addition["frontend"].(map[string]interface{}); ok {
			c.parseFrontendData(frontendObj, timestampData)
		}
		return timestampData, duration
	}

	// 解析 JSON 字符串
	var frontendData map[string]interface{}
	if err := json.Unmarshal([]byte(frontendStr), &frontendData); err != nil {
		log.Warn().Err(err).Msg("failed to parse frontend data")
		return timestampData, duration
	}

	c.parseFrontendData(frontendData, timestampData)

	return timestampData, duration
}

// parseFrontendData 解析前端数据中的时间戳
func (c *Client) parseFrontendData(frontendData map[string]interface{}, timestampData *TimestampData) {
	words, ok := frontendData["words"].([]interface{})
	if !ok {
		return
	}

	var charTimestamps []CharTimestamp
	for _, wordItem := range words {
		wordInfo, ok := wordItem.(map[string]interface{})
		if !ok {
			continue
		}

		word, _ := wordInfo["word"].(string)
		startTime, _ := wordInfo["start_time"].(float64)
		endTime, _ := wordInfo["end_time"].(float64)

		// 将词级时间戳转换为字符级时间戳（简化处理）
		// 实际 API 可能直接返回字符级时间戳
		if word != "" {
			charCount := len([]rune(word))
			if charCount > 0 {
				duration := endTime - startTime
				charDuration := duration / float64(charCount)

				for i, char := range []rune(word) {
					charStartTime := startTime + float64(i)*charDuration
					charEndTime := startTime + float64(i+1)*charDuration

					charTimestamps = append(charTimestamps, CharTimestamp{
						Character: string(char),
						StartTime: charStartTime,
						EndTime:   charEndTime,
					})
				}
			}
		}
	}

	timestampData.CharacterTimestamps = charTimestamps
}

// fixJSON 修复 JSON 字符串（参考 Python 代码的修复逻辑）
func (c *Client) fixJSON(jsonStr string) string {
	// 策略1: 修复缺少逗号的问题
	fixed := jsonStr
	fixed = strings.ReplaceAll(fixed, "}{", "},{")

	// 策略2: 修复字符串后缺少逗号的问题
	fixed = strings.ReplaceAll(fixed, "\"}{\"", "\"},{\"")

	// 策略3: 修复特定模式
	fixed = strings.ReplaceAll(fixed, "}{\"phone", "},{\"phone")
	fixed = strings.ReplaceAll(fixed, "}{\"word", "},{\"word")

	return fixed
}

// requestBodyReader 用于支持多次读取请求体
type requestBodyReader struct {
	data []byte
	pos  int
}

func (r *requestBodyReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
