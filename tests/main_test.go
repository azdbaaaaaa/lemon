// Package tests 集成测试
//
// 运行集成测试：
//
//	MONGO_URI=mongodb://localhost:27017 go test ./tests -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - KEEP_TEST_DATA: 设置为 "true" 时，测试完成后保留数据库数据和存储文件（默认: false，会自动清理）
//   - 测试使用本地文件系统存储（临时目录）
//   - 测试完成后默认会自动清理测试数据库和临时存储文件
package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/config"
	"lemon/internal/pkg/storage"
	"lemon/internal/pkg/storagefactory"
)

// getTestStorageDir 获取测试存储目录
func getTestStorageDir(t *testing.T) string {
	// 获取项目根目录
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	// 向上找到项目根目录（从 tests 目录到项目根）
	for !strings.HasSuffix(projectRoot, "lemon") && len(projectRoot) > 1 {
		projectRoot = filepath.Dir(projectRoot)
	}
	if !strings.HasSuffix(projectRoot, "lemon") {
		t.Fatalf("Failed to find project root")
	}

	// 使用 tmp 目录作为测试存储路径
	return filepath.Join(projectRoot, "tmp", "integration_test_storage")
}

// getTestNovelFilePath 获取测试小说文件路径
func getTestNovelFilePath(t *testing.T) string {
	// 获取项目根目录
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	// 向上找到项目根目录
	for !strings.HasSuffix(projectRoot, "lemon") && len(projectRoot) > 1 {
		projectRoot = filepath.Dir(projectRoot)
	}
	if !strings.HasSuffix(projectRoot, "lemon") {
		t.Fatalf("Failed to find project root")
	}

	// 构建测试文件路径
	novelFilePath := filepath.Join(projectRoot, "assets", "novel", "001", "大道主(飘荡的云).txt")

	// 如果文件不存在，尝试从 tests 目录查找
	if _, err := os.Stat(novelFilePath); os.IsNotExist(err) {
		novelFilePath = filepath.Join(projectRoot, "..", "assets", "novel", "001", "大道主(飘荡的云).txt")
	}

	return novelFilePath
}

// setupTestEnvironment 设置测试环境（MongoDB 和存储）
func setupTestEnvironment(t *testing.T) (context.Context, *mongo.Database, storage.Storage, func()) {
	ctx := context.Background()

	// 1. 初始化 MongoDB 连接（使用测试数据库）
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	So(err, ShouldBeNil)

	// 使用测试数据库
	db := mongoClient.Database("lemon_test")

	// 2. 初始化存储（本地文件系统存储）
	testStorageDir := getTestStorageDir(t)
	storageCfg := &config.StorageConfig{
		Type: "local",
		Local: &config.LocalConfig{
			BasePath:      testStorageDir,
			BaseURL:       "http://localhost:7080/storage",
			PresignExpiry: 3600,
		},
	}

	testStorage, err := storagefactory.NewStorage(ctx, storageCfg)
	So(err, ShouldBeNil)
	So(testStorage, ShouldNotBeNil)

	// 检查是否保留测试数据
	keepTestData := os.Getenv("KEEP_TEST_DATA") == "true"

	// 清理函数
	cleanup := func() {
		if !keepTestData {
			// 清理数据库集合
			_ = db.Collection("resources").Drop(ctx)
			_ = db.Collection("upload_sessions").Drop(ctx)
			_ = db.Collection("novels").Drop(ctx)
			_ = db.Collection("chapters").Drop(ctx)
			// 清理存储文件
			_ = os.RemoveAll(testStorageDir)
		} else {
			// 保留数据，只断开连接
			t.Logf("保留测试数据：数据库=%s, 存储目录=%s", db.Name(), testStorageDir)
		}
		_ = mongoClient.Disconnect(ctx)
	}

	return ctx, db, testStorage, cleanup
}
