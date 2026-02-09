package novel

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// SceneRepository 场景仓库接口
type SceneRepository interface {
	Create(ctx context.Context, scene *novel.Scene) error
	CreateMany(ctx context.Context, scenes []*novel.Scene) error
	FindByID(ctx context.Context, id string) (*novel.Scene, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Scene, error)
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.Scene, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error
	Delete(ctx context.Context, id string) error
	DeleteByNovelID(ctx context.Context, novelID string) error
}

// SceneRepo 场景仓库实现
type SceneRepo struct {
	coll *mongo.Collection
}

// NewSceneRepo 创建场景仓库
func NewSceneRepo(db *mongo.Database) *SceneRepo {
	var s novel.Scene
	return &SceneRepo{coll: db.Collection(s.Collection())}
}

// Create 创建场景
func (r *SceneRepo) Create(ctx context.Context, scene *novel.Scene) error {
	now := time.Now()
	scene.CreatedAt = now
	scene.UpdatedAt = now
	if scene.Status == "" || scene.Status == novel.TaskStatus("") {
		scene.Status = novel.TaskStatusCompleted
	}
	_, err := r.coll.InsertOne(ctx, scene)
	return err
}

// CreateMany 批量创建场景
func (r *SceneRepo) CreateMany(ctx context.Context, scenes []*novel.Scene) error {
	if len(scenes) == 0 {
		return nil
	}
	now := time.Now()
	docs := make([]interface{}, len(scenes))
	for i, scene := range scenes {
		scene.CreatedAt = now
		scene.UpdatedAt = now
		if scene.Status == "" {
			scene.Status = novel.TaskStatusCompleted
		}
		docs[i] = scene
	}
	_, err := r.coll.InsertMany(ctx, docs)
	return err
}

// FindByID 根据ID查询场景
func (r *SceneRepo) FindByID(ctx context.Context, id string) (*novel.Scene, error) {
	var scene novel.Scene
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&scene); err != nil {
		return nil, err
	}
	return &scene, nil
}

// FindByChapterID 根据章节ID查询所有场景（查询所有版本）
func (r *SceneRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Scene, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.D{
		{Key: "version", Value: -1},
		{Key: "sequence", Value: 1},
	})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		log.Error().Err(err).
			Str("chapter_id", chapterID).
			Msg("查询场景列表失败")
		return nil, err
	}
	defer cur.Close(ctx)

	var scenes []*novel.Scene
	if err := cur.All(ctx, &scenes); err != nil {
		log.Error().Err(err).
			Str("chapter_id", chapterID).
			Msg("解析场景列表失败")
		return nil, err
	}
	return scenes, nil
}

// FindByChapterIDAndVersion 根据章节ID和版本号查询场景
func (r *SceneRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.Scene, error) {
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		log.Error().Err(err).
			Str("chapter_id", chapterID).
			Int("version", version).
			Msg("按版本查询场景列表失败")
		return nil, err
	}
	defer cur.Close(ctx)

	var scenes []*novel.Scene
	if err := cur.All(ctx, &scenes); err != nil {
		log.Error().Err(err).
			Str("chapter_id", chapterID).
			Int("version", version).
			Msg("解析场景列表失败")
		return nil, err
	}
	return scenes, nil
}

// Update 更新场景
func (r *SceneRepo) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": updates},
	)
	return err
}

// UpdateStatus 更新场景状态
func (r *SceneRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error {
	update := bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errorMessage != "" {
		update["error_message"] = errorMessage
	} else {
		update["$unset"] = bson.M{"error_message": ""}
	}
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": update},
	)
	return err
}

// Delete 软删除场景
func (r *SceneRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)
	return err
}

// DeleteByNovelID 根据小说ID删除所有场景（软删除）
func (r *SceneRepo) DeleteByNovelID(ctx context.Context, novelID string) error {
	_, err := r.coll.UpdateMany(
		ctx,
		bson.M{"novel_id": novelID, "deleted_at": nil},
		bson.M{"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)
	return err
}
