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
//   - 注意：如果所有集合都被删除，MongoDB 可能会自动删除空数据库，导致看不到数据库
//   - 建议：使用 KEEP_TEST_DATA=true 来保留测试数据，或使用 MongoDB 客户端手动查看数据
package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/config"
	"lemon/internal/pkg/storage"
	"lemon/internal/pkg/storagefactory"
	"lemon/internal/service"
	novelservice "lemon/internal/service/novel"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
)

// 包级别的测试环境变量（在 TestMain 中初始化）
var (
	testCtx         context.Context
	testDB          *mongo.Database
	testStorage     storage.Storage
	testStorageDir  string
	testServices    *TestServices
	testCleanup     func()
	testMongoClient *mongo.Client
)

// TestMain 测试主函数，在所有测试运行前初始化和运行后清理
func TestMain(m *testing.M) {
	// 初始化测试环境
	testCtx = context.Background()

	// 1. 初始化 MongoDB 连接（使用测试数据库）
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	var err error
	testMongoClient, err = mongo.Connect(testCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MongoDB: %v", err))
	}

	// 使用测试数据库
	testDB = testMongoClient.Database("lemon_test")

	// 2. 初始化存储（本地文件系统存储）
	testStorageDir = getTestStorageDirForMain()
	storageCfg := &config.StorageConfig{
		Type: "local",
		Local: &config.LocalConfig{
			BasePath:      testStorageDir,
			BaseURL:       "http://localhost:7080/storage",
			PresignExpiry: 3600,
		},
	}

	var err2 error
	testStorage, err2 = storagefactory.NewStorage(testCtx, storageCfg)
	if err2 != nil {
		panic(fmt.Sprintf("Failed to create storage: %v", err2))
	}

	// 3. 初始化测试服务（providers 现在由 NovelService 内部管理）
	testServices = setupTestServices(testDB, testStorage)

	// 6. 设置清理函数
	keepTestDataEnv := os.Getenv("KEEP_TEST_DATA")
	keepTestData := keepTestDataEnv == "true"
	fmt.Fprintf(os.Stderr, "[TestMain] KEEP_TEST_DATA 环境变量值: %q, keepTestData=%v\n", keepTestDataEnv, keepTestData)
	testCleanup = func() {
		if !keepTestData {
			// 清理数据库集合（按顺序删除，避免依赖问题）
			// 注意：删除集合不会删除数据库本身，但如果所有集合都被删除，MongoDB 可能会在下次访问时自动删除空数据库
			collections := []string{
				"scene_shot_images", // 先删除图片
				"subtitles",         // 删除字幕
				"audios",            // 删除音频
				"characters",        // 删除角色
				"narrations",        // 删除解说文案
				"chapters",          // 删除章节
				"novels",            // 删除小说
				"upload_sessions",   // 删除上传会话
				"resources",         // 最后删除资源
			}
			for _, collName := range collections {
				if err := testDB.Collection(collName).Drop(testCtx); err != nil {
					// 集合不存在时忽略错误
					_ = err
				}
			}
			// 清理存储文件
			_ = os.RemoveAll(testStorageDir)
			fmt.Fprintf(os.Stderr, "[TestMain] 已清理测试数据：数据库=%s, 存储目录=%s\n", testDB.Name(), testStorageDir)
		} else {
			// 保留数据，只记录日志（使用 os.Stderr 确保输出可见）
			fmt.Fprintf(os.Stderr, "[TestMain] 保留测试数据：数据库=%s, 存储目录=%s\n", testDB.Name(), testStorageDir)
			fmt.Fprintf(os.Stderr, "[TestMain] 提示：使用 MongoDB 客户端连接查看数据，数据库名称: %s\n", testDB.Name())
		}
		_ = testMongoClient.Disconnect(testCtx)
	}

	// 运行所有测试
	code := m.Run()

	// 清理资源
	testCleanup()

	// 退出
	os.Exit(code)
}

// getTestStorageDirForMain 获取测试存储目录（用于 TestMain，不需要 testing.T）
func getTestStorageDirForMain() string {
	// 获取项目根目录
	projectRoot, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Failed to get current directory: %v", err))
	}
	// 向上找到项目根目录（从 tests 目录到项目根）
	for !strings.HasSuffix(projectRoot, "lemon") && len(projectRoot) > 1 {
		projectRoot = filepath.Dir(projectRoot)
	}
	if !strings.HasSuffix(projectRoot, "lemon") {
		panic("Failed to find project root")
	}

	// 使用 tmp 目录作为测试存储路径
	return filepath.Join(projectRoot, "tmp", "integration_test_storage")
}

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

	// 检查是否保留测试数据
	keepTestData := os.Getenv("KEEP_TEST_DATA") == "true"

	// 清理函数
	cleanup := func() {
		if !keepTestData {
			// 清理数据库集合（按顺序删除，避免依赖问题）
			collections := []string{
				"subtitles",
				"audios",
				"narrations",
				"chapters",
				"novels",
				"upload_sessions",
				"resources",
			}
			for _, collName := range collections {
				if err := db.Collection(collName).Drop(ctx); err != nil {
					// 集合不存在时忽略错误
					_ = err
				}
			}
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

// uploadTestFile 上传测试文件并返回资源ID（使用服务端上传方式）
// 这是一个辅助函数，用于在后续测试中直接使用已上传的资源
func uploadTestFile(ctx context.Context, t *testing.T, resourceService service.ResourceService, userID string) string {
	// 读取测试文件
	novelFilePath := getTestNovelFilePath(t)
	novelFile, err := os.Open(novelFilePath)
	if err != nil {
		t.Fatalf("应该能打开测试文件: %s, 错误: %v", novelFilePath, err)
	}
	defer novelFile.Close()

	fileStat, err := novelFile.Stat()
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	// 重置文件指针到开头
	_, err = novelFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("重置文件指针失败: %v", err)
	}

	// 使用服务端上传方式（UploadFile）
	uploadReq := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    fileStat.Name(),
		ContentType: "text/plain",
		Ext:         "txt",
		Data:        novelFile,
	}

	uploadResult, err := resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		t.Fatalf("上传文件失败: %v", err)
	}

	return uploadResult.ResourceID
}

// findOrCreateTestChapters 查找或创建测试章节
// 优先查找数据库中已有的章节，如果没有找到则创建新的章节
func findOrCreateTestChapters(ctx context.Context, t *testing.T, services *TestServices, userID string) (string, []*novel.Chapter) {
	// 1. 先尝试查找已有的章节（直接通过 userID 查找章节）
	// 直接查询章节集合，查找该用户的最新章节
	var chapterModel novel.Chapter
	coll := testDB.Collection(chapterModel.Collection())

	filter := bson.M{"user_id": userID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(1)
	cursor, err := coll.Find(ctx, filter, opts)
	if err == nil {
		var foundChapters []*novel.Chapter
		if err := cursor.All(ctx, &foundChapters); err == nil && len(foundChapters) > 0 {
			// 找到了章节，获取该章节所属的小说ID
			firstChapter := foundChapters[0]
			novelID := firstChapter.NovelID

			// 获取该小说的所有章节
			chapters, err := services.NovelService.GetChapters(ctx, novelID)
			if err == nil && len(chapters) > 0 {
				// 验证章节是否有内容
				hasContent := false
				for _, ch := range chapters {
					if ch.ChapterText != "" {
						hasContent = true
						break
					}
				}
				if hasContent {
					t.Logf("使用数据库中已有的章节: 小说ID=%s, 章节数=%d", novelID, len(chapters))
					return novelID, chapters
				}
			}
		}
		cursor.Close(ctx)
	}

	// 2. 如果没有找到，则创建新的章节
	t.Logf("未找到可用的章节，开始创建新章节...")
	workflowID := id.New()
	resourceID := findOrUploadTestFile(ctx, t, services, userID)

	novelID, err := services.NovelService.CreateNovelFromResource(ctx, resourceID, userID, workflowID)
	if err != nil {
		t.Fatalf("创建小说失败: %v", err)
	}

	targetChapters := 5
	err = services.NovelService.SplitNovelIntoChapters(ctx, novelID, targetChapters)
	if err != nil {
		t.Fatalf("切分章节失败: %v", err)
	}

	chapters, err := services.NovelService.GetChapters(ctx, novelID)
	if err != nil {
		t.Fatalf("查询章节失败: %v", err)
	}

	return novelID, chapters
}

// findOrCreateTestNarration 查找或创建测试解说文案
// 优先使用数据库中已有的解说文案（从已有的章节中查找）
func findOrCreateTestNarration(ctx context.Context, t *testing.T, services *TestServices, userID string) (string, *novel.Narration) {
	// 1. 先尝试查找已有的章节
	_, chapters := findOrCreateTestChapters(ctx, t, services, userID)
	if len(chapters) == 0 {
		t.Fatal("无法找到或创建测试章节")
	}

	// 2. 查找第一个章节的解说文案
	firstChapter := chapters[0]
	narrationEntity, err := services.NovelService.GetNarration(ctx, firstChapter.ID)
	if err == nil {
		// 找到了已有的解说文案
		return narrationEntity.ID, narrationEntity
	}

	// 3. 如果没有找到，尝试生成一个（需要 LLM Provider）
	// 如果 TestMain 成功执行，LLMProvider 一定已初始化
	// 4. 生成解说文案
	narrationText, err := services.NovelService.GenerateNarrationForChapter(ctx, firstChapter.ID)
	if err != nil {
		t.Fatalf("生成解说文案失败: %v", err)
	}
	if narrationText == "" {
		t.Fatal("生成的解说文案为空")
	}

	// 5. 再次查询，获取生成的解说文案
	narrationEntity, err = services.NovelService.GetNarration(ctx, firstChapter.ID)
	if err != nil {
		t.Fatalf("查询生成的解说文案失败: %v", err)
	}

	return narrationEntity.ID, narrationEntity
}

// findOrUploadTestFile 查找或上传测试文件
// 优先查找数据库中已有的资源，如果没有找到再上传
func findOrUploadTestFile(ctx context.Context, t *testing.T, services *TestServices, userID string) string {
	// 1. 先尝试查找数据库中已有的资源（按创建时间降序，取最新的）
	listResult, err := services.ResourceService.ListResources(ctx, &service.ListResourcesRequest{
		UserID:   userID,
		Page:     1,
		PageSize: 1,
	})
	if err == nil && len(listResult.Resources) > 0 {
		// 找到了已有的资源，直接使用
		resource := listResult.Resources[0]
		// 验证资源状态是 ready
		if resource.Status == "ready" {
			t.Logf("使用数据库中已有的资源: %s (文件名: %s)", resource.ID, resource.Name)
			return resource.ID
		}
	}

	// 2. 如果没有找到或资源状态不对，则上传新文件
	t.Logf("未找到可用的资源，开始上传新文件...")
	return uploadTestFile(ctx, t, services.ResourceService, userID)
}

// TestServices 测试服务集合
// 包含所有测试中需要的服务
// 注意：测试应该使用 Service 层，不应该直接使用 Repository 层
type TestServices struct {
	// 服务
	ResourceService service.ResourceService
	NovelService    novelservice.NovelService

	// 存储
	Storage storage.Storage

	// 注意：Providers 现在由 NovelService 内部管理，不再需要单独传入
}

// setupTestServices 初始化测试服务
// Providers 现在由 NovelService 内部管理，不再需要单独传入
// 测试应该使用 Service 层，不需要直接访问 Repository 层
func setupTestServices(db *mongo.Database, testStorage storage.Storage) *TestServices {
	// 初始化 ResourceService（内部自动创建 repository）
	resourceService := service.NewResourceService(db, testStorage)

	// 初始化 NovelService（内部自动创建所有 repository）
	novelService, err := novelservice.NewNovelService(
		db,
		resourceService,
	)
	if err != nil {
		panic(fmt.Sprintf("初始化 NovelService 失败: %v", err))
	}

	return &TestServices{
		ResourceService: resourceService,
		NovelService:    novelService,
		Storage:         testStorage,
	}
}
