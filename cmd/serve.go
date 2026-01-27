package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"lemon/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Lemon API server with the specified configuration.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	flags := serveCmd.Flags()

	// Server flags
	flags.StringP("host", "H", "0.0.0.0", "server host")
	flags.IntP("port", "p", 7080, "server port")
	flags.String("mode", "release", "server mode (debug/release/test)")

	// AI flags
	flags.String("ai-provider", "openai", "AI provider (openai/azure/anthropic)")
	flags.String("ai-model", "gpt-4", "AI model name")
	flags.String("ai-api-key", "", "AI API key (recommend using env: LEMON_AI_API_KEY)")

	// Log flags
	flags.String("log-level", "info", "log level (trace/debug/info/warn/error/fatal)")
	flags.String("log-format", "console", "log format (json/console)")

	// Bind flags to viper
	_ = viper.BindPFlag("server.host", flags.Lookup("host"))
	_ = viper.BindPFlag("server.port", flags.Lookup("port"))
	_ = viper.BindPFlag("server.mode", flags.Lookup("mode"))
	_ = viper.BindPFlag("ai.provider", flags.Lookup("ai-provider"))
	_ = viper.BindPFlag("ai.model", flags.Lookup("ai-model"))
	_ = viper.BindPFlag("ai.api_key", flags.Lookup("ai-api-key"))
	_ = viper.BindPFlag("log.level", flags.Lookup("log-level"))
	_ = viper.BindPFlag("log.format", flags.Lookup("log-format"))
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("received shutdown signal")
		cancel()
	}()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info().
		Str("addr", addr).
		Str("mode", cfg.Server.Mode).
		Msg("starting server")

	return srv.Run(ctx, addr)
}
