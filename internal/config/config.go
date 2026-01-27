package config

import (
	"errors"
	"time"
)

// Config 应用配置根结构
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	AI      AIConfig      `mapstructure:"ai"`
	Log     LogConfig     `mapstructure:"log"`
	Mongo   MongoConfig   `mapstructure:"mongo"`
	Redis   RedisConfig   `mapstructure:"redis"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Storage StorageConfig `mapstructure:"storage"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// AIConfig AI 服务配置
type AIConfig struct {
	Provider string          `mapstructure:"provider"`
	APIKey   string          `mapstructure:"api_key"`
	Model    string          `mapstructure:"model"`
	BaseURL  string          `mapstructure:"base_url"`
	Options  AIOptionsConfig `mapstructure:"options"`
}

// AIOptionsConfig AI 模型参数
type AIOptionsConfig struct {
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	TopP        float64 `mapstructure:"top_p"`
}

// LogConfig 日志配置 (Zerolog)
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	TimeFormat string `mapstructure:"time_format"`
}

// MongoConfig MongoDB 配置
type MongoConfig struct {
	URI         string `mapstructure:"uri"`
	Database    string `mapstructure:"database"`
	MaxPoolSize uint64 `mapstructure:"max_pool_size"`
	MinPoolSize uint64 `mapstructure:"min_pool_size"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"`           // JWT密钥
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`  // Access Token过期时间
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"` // Refresh Token过期时间
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type  string       `mapstructure:"type"` // local, oss, s3, minio
	Local *LocalConfig `mapstructure:"local,omitempty"`
	OSS   *OSSConfig   `mapstructure:"oss,omitempty"`
}

// LocalConfig 本地文件系统配置
type LocalConfig struct {
	BasePath      string `mapstructure:"base_path"`      // 基础路径
	BaseURL       string `mapstructure:"base_url"`       // 基础URL（用于生成访问URL）
	PresignExpiry int    `mapstructure:"presign_expiry"` // 预签名URL过期时间（秒）
}

// OSSConfig 阿里云OSS配置
type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`          // OSS端点
	Bucket          string `mapstructure:"bucket"`            // Bucket名称
	AccessKeyID     string `mapstructure:"access_key_id"`     // AccessKey ID
	AccessKeySecret string `mapstructure:"access_key_secret"` // AccessKey Secret
	PresignExpiry   int    `mapstructure:"presign_expiry"`    // 预签名URL过期时间（秒）
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return errors.New("invalid server port")
	}

	validModes := map[string]bool{"debug": true, "release": true, "test": true}
	if !validModes[c.Server.Mode] {
		return errors.New("invalid server mode, must be debug/release/test")
	}

	return nil
}
