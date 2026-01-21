package config

import (
	"errors"
	"time"
)

// Config 应用配置根结构
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	AI     AIConfig     `mapstructure:"ai"`
	Log    LogConfig    `mapstructure:"log"`
	Mongo  MongoConfig  `mapstructure:"mongo"`
	Redis  RedisConfig  `mapstructure:"redis"`
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
