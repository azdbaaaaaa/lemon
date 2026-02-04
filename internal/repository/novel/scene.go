package novel

import (
	"context"
	"time"

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
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Scene, error)
	FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Scene, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Scene, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	DeleteByNarrationID(ctx context.Context, narrationID string) error
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
	if scene.Status == "" {
		scene.Status = "completed"
	}
	if scene.Version == 0 {
		scene.Version = 1
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
			scene.Status = "completed"
		}
		if scene.Version == 0 {
			scene.Version = 1
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

// FindByNarrationID 根据解说ID查询所有场景（按sequence排序）
func (r *SceneRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Scene, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var scenes []*novel.Scene
	if err := cur.All(ctx, &scenes); err != nil {
		return nil, err
	}
	return scenes, nil
}

// FindByNarrationIDAndVersion 根据解说ID和版本号查询场景
func (r *SceneRepo) FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Scene, error) {
	filter := bson.M{"narration_id": narrationID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var scenes []*novel.Scene
	if err := cur.All(ctx, &scenes); err != nil {
		return nil, err
	}
	return scenes, nil
}

// FindByChapterID 根据章节ID查询所有场景
func (r *SceneRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Scene, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var scenes []*novel.Scene
	if err := cur.All(ctx, &scenes); err != nil {
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

// DeleteByNarrationID 根据解说ID软删除所有场景
func (r *SceneRepo) DeleteByNarrationID(ctx context.Context, narrationID string) error {
	_, err := r.coll.UpdateMany(
		ctx,
		bson.M{"narration_id": narrationID, "deleted_at": nil},
		bson.M{"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)
	return err
}
