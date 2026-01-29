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
	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/pkg/noveltools/providers"
	"lemon/internal/pkg/storage"
	"lemon/internal/pkg/storagefactory"
	narrationRepo "lemon/internal/repository/narration"
	novelrepo "lemon/internal/repository/novel"
	resourceRepo "lemon/internal/repository/resource"
	"lemon/internal/service"

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

	// 3. 初始化 LLM Provider（从环境变量读取配置）
	var llmProvider noveltools.LLMProvider
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey != "" {
		aiCfg := &config.AIConfig{
			Provider: "ark",
			APIKey:   apiKey,
			Model:    os.Getenv("ARK_MODEL"),
			BaseURL:  os.Getenv("ARK_BASE_URL"),
		}
		if aiCfg.Model == "" {
			aiCfg.Model = "doubao-seed-1-6-flash-250615"
		}
		if aiCfg.BaseURL == "" {
			aiCfg.BaseURL = "https://ark.cn-beijing.volces.com/api/v3"
		}

		arkClient, err := ark.NewClient(aiCfg)
		if err == nil {
			llmProvider = providers.NewArkProvider(arkClient)
			fmt.Fprintf(os.Stderr, "已初始化真实的 LLM Provider (Ark)\n")
		} else {
			fmt.Fprintf(os.Stderr, "警告: 初始化 LLM Provider 失败: %v，将使用 nil\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "警告: ARK_API_KEY 未设置，LLM Provider 为 nil（某些测试可能需要）\n")
	}

	// 4. 初始化测试服务
	testServices = setupTestServices(testDB, testStorage, llmProvider)

	// 5. 设置清理函数
	keepTestData := os.Getenv("KEEP_TEST_DATA") == "true"
	testCleanup = func() {
		if !keepTestData {
			// 清理数据库集合
			_ = testDB.Collection("resources").Drop(testCtx)
			_ = testDB.Collection("upload_sessions").Drop(testCtx)
			_ = testDB.Collection("novels").Drop(testCtx)
			_ = testDB.Collection("chapters").Drop(testCtx)
			_ = testDB.Collection("narrations").Drop(testCtx)
			// 清理存储文件
			_ = os.RemoveAll(testStorageDir)
		} else {
			// 保留数据，只记录日志（使用 os.Stderr 确保输出可见）
			fmt.Fprintf(os.Stderr, "保留测试数据：数据库=%s, 存储目录=%s\n", testDB.Name(), testStorageDir)
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

// uploadTestFile 上传测试文件并返回资源ID（复用第一步上传流程）
// 这是一个辅助函数，用于在后续测试中直接使用已上传的资源
func uploadTestFile(ctx context.Context, t *testing.T, resourceService *service.ResourceService, testStorage storage.Storage, userID string) string {
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

	// 准备上传
	prepareReq := &service.PrepareUploadRequest{
		UserID:      userID,
		FileName:    fileStat.Name(),
		FileSize:    fileStat.Size(),
		ContentType: "text/plain",
		Ext:         "txt",
	}

	prepareResult, err := resourceService.PrepareUpload(ctx, prepareReq)
	if err != nil {
		t.Fatalf("准备上传失败: %v", err)
	}

	// 上传文件
	_, err = novelFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("重置文件指针失败: %v", err)
	}

	_, err = testStorage.Upload(ctx, prepareResult.UploadKey, novelFile, "text/plain")
	if err != nil {
		t.Fatalf("上传文件失败: %v", err)
	}

	// 完成上传
	completeReq := &service.CompleteUploadRequest{
		SessionID: prepareResult.SessionID,
	}

	completeResult, err := resourceService.CompleteUpload(ctx, completeReq)
	if err != nil {
		t.Fatalf("完成上传失败: %v", err)
	}

	return completeResult.ResourceID
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
			chapters, err := services.ChapterRepo.FindByNovelID(ctx, novelID)
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

	chapters, err := services.ChapterRepo.FindByNovelID(ctx, novelID)
	if err != nil {
		t.Fatalf("查询章节失败: %v", err)
	}

	return novelID, chapters
}

// findOrUploadTestFile 查找或上传测试文件
// 优先查找数据库中已有的资源，如果没有找到再上传
func findOrUploadTestFile(ctx context.Context, t *testing.T, services *TestServices, userID string) string {
	// 1. 先尝试查找数据库中已有的资源（按创建时间降序，取最新的）
	resources, _, err := services.ResourceRepo.FindByUserID(ctx, userID, 1, 0)
	if err == nil && len(resources) > 0 {
		// 找到了已有的资源，直接使用
		resource := resources[0]
		// 验证资源状态是 ready
		if resource.Status == "ready" {
			t.Logf("使用数据库中已有的资源: %s (文件名: %s)", resource.ID, resource.Name)
			return resource.ID
		}
	}

	// 2. 如果没有找到或资源状态不对，则上传新文件
	t.Logf("未找到可用的资源，开始上传新文件...")
	return uploadTestFile(ctx, t, services.ResourceService, services.Storage, userID)
}

// TestServices 测试服务集合
// 包含所有测试中需要的仓库和服务
type TestServices struct {
	// 仓库
	ResourceRepo  *resourceRepo.ResourceRepo
	NovelRepo     novelrepo.NovelRepository
	ChapterRepo   novelrepo.ChapterRepository
	NarrationRepo narrationRepo.NarrationRepository

	// 服务
	ResourceService *service.ResourceService
	NovelService    *service.NovelService

	// 存储
	Storage storage.Storage

	// LLM Provider（用于生成解说文案）
	LLMProvider noveltools.LLMProvider
}

// setupTestServices 初始化测试服务（仓库和服务）
func setupTestServices(db *mongo.Database, testStorage storage.Storage, llmProvider noveltools.LLMProvider) *TestServices {
	resourceRepo := resourceRepo.NewResourceRepo(db)
	novelRepo := novelrepo.NewNovelRepo(db)
	chapterRepo := novelrepo.NewChapterRepo(db)
	narrationRepo := narrationRepo.NewNarrationRepo(db)

	resourceService := service.NewResourceService(resourceRepo, testStorage)
	novelService := service.NewNovelService(resourceRepo, novelRepo, chapterRepo, narrationRepo, testStorage, llmProvider)

	return &TestServices{
		ResourceRepo:    resourceRepo,
		NovelRepo:       novelRepo,
		ChapterRepo:     chapterRepo,
		NarrationRepo:   narrationRepo,
		ResourceService: resourceService,
		NovelService:    novelService,
		Storage:         testStorage,
		LLMProvider:     llmProvider,
	}
}
