package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"lemon/internal/config"
	"lemon/internal/pkg/logger"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "lemon",
	Short: "Lemon - AI-powered API Service",
	Long: `Lemon is an AI-powered API service built with Eino framework.
It provides LLM chat, intelligent agent, and more AI capabilities.`,
	SilenceUsage: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
		"config file (default: ./configs/config.yaml)")

	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.lemon")
	}

	// 环境变量设置
	viper.SetEnvPrefix("LEMON")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			fmt.Fprintln(os.Stderr, "No config file found, using defaults and environment variables")
		} else {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
			os.Exit(1)
		}
	}

	// 反序列化到结构体
	cfg = &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	log.Debug().Str("config_file", viper.ConfigFileUsed()).Msg("configuration loaded")
}

func setDefaults() {
	// Server
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")

	// AI
	viper.SetDefault("ai.provider", "openai")
	viper.SetDefault("ai.model", "gpt-4")
	viper.SetDefault("ai.options.temperature", 0.7)
	viper.SetDefault("ai.options.max_tokens", 4096)
	viper.SetDefault("ai.options.top_p", 1.0)

	// Log
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.time_format", "RFC3339")

	// MongoDB
	viper.SetDefault("mongo.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongo.database", "lemon")
	viper.SetDefault("mongo.max_pool_size", 100)
	viper.SetDefault("mongo.min_pool_size", 10)

	// Redis
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}
