package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"lemon/internal/config"
	"lemon/internal/model/auth"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/logger"
	"lemon/internal/pkg/mongodb"
	"lemon/internal/pkg/password"
	authrepo "lemon/internal/repository/auth"
)

func main() {
	// 1. 加载配置（与 cmd/root.go 保持一致的搜索路径）
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.lemon")

	viper.SetEnvPrefix("LEMON")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
		os.Exit(1)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	// 2. 连接 MongoDB
	client, err := mongodb.New(&cfg.Mongo)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect mongo")
	}
	defer func() {
		_ = client.Close(context.Background())
	}()

	db := client.Database()
	ctx := context.Background()

	// 3. 初始化 UserRepo
	userRepo := authrepo.NewUserRepo(db)

	// 4. 读取环境变量或使用默认值
	username := os.Getenv("INIT_ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	passwordPlain := os.Getenv("INIT_ADMIN_PASSWORD")
	if passwordPlain == "" {
		passwordPlain = "admin123"
	}
	email := os.Getenv("INIT_ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.com"
	}

	// 5. 检查是否已存在
	user, err := userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Info().Str("username", username).Msg("admin user not found, will create")
			if err := createAdmin(ctx, userRepo, username, email, passwordPlain); err != nil {
				log.Fatal().Err(err).Msg("create admin user failed")
			}
		} else {
			log.Fatal().Err(err).Msg("failed to query user")
		}
	} else {
		// 已存在，更新为 admin + active
		log.Info().Str("username", username).Msg("admin user exists, will update role/status")
		update := bson.M{
			"$set": bson.M{
				"role":   auth.RoleAdmin,
				"status": auth.UserStatusActive,
				"email":  email,
			},
		}
		if err := userRepo.Update(ctx, user.ID, update); err != nil {
			log.Fatal().Err(err).Msg("update admin user failed")
		}
	}

	fmt.Printf("Admin initialized: username=%s password=%s role=admin status=active\n",
		username, passwordPlain)
}

func createAdmin(ctx context.Context, repo *authrepo.UserRepo, username, email, pwd string) error {
	hashed, err := password.Hash(pwd)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &auth.User{
		ID:       id.New(),
		Username: username,
		Email:    email,
		Password: hashed,
		Role:     auth.RoleAdmin,
		Status:   auth.UserStatusActive,
		Profile: &auth.UserProfile{
			Nickname: "管理员",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	return repo.Create(ctx, user)
}


