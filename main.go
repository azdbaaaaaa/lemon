package main

import (
	"os"

	"lemon/cmd"
	_ "lemon/docs/swagger" // swagger docs
)

// @title           Lemon API
// @version         1.0
// @description     Lemon is an AI-powered API service built with Eino framework.
// @description     It provides LLM chat, intelligent agent, and more AI capabilities.

// @contact.name   API Support
// @contact.email  support@lemon.ai

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @schemes   http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
