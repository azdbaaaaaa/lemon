package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// SceneShotImageRepository 场景特写图片仓库接口（供 service 层依赖）
type SceneShotImageRepository interface {
	Create(ctx context.Context, image *novel.SceneShotImage) error
	FindByID(ctx context.Context, id string) (*novel.SceneShotImage, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.SceneShotImage, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.SceneShotImage, error)
	FindBySceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.SceneShotImage, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	Delete(ctx context.Context, id string) error
}

// SceneShotImageRepo 场景特写图片仓库
type SceneShotImageRepo struct {
	coll *mongo.Collection
}

// NewSceneShotImageRepo 创建场景特写图片仓库
func NewSceneShotImageRepo(db *mongo.Database) *SceneShotImageRepo {
	var s novel.SceneShotImage
	return &SceneShotImageRepo{coll: db.Collection(s.Collection())}
}

// Create 创建图片记录
func (r *SceneShotImageRepo) Create(ctx context.Context, image *novel.SceneShotImage) error {
	now := time.Now()
	image.CreatedAt = now
	image.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, image)
	return err
}

// FindByID 根据ID查询
func (r *SceneShotImageRepo) FindByID(ctx context.Context, id string) (*novel.SceneShotImage, error) {
	var image novel.SceneShotImage
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

// FindByChapterID 查询章节的所有图片
func (r *SceneShotImageRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.SceneShotImage, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.SceneShotImage
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByNarrationID 查询解说文案的所有图片
func (r *SceneShotImageRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.SceneShotImage, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.SceneShotImage
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindBySceneAndShot 根据场景和特写编号查询
func (r *SceneShotImageRepo) FindBySceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.SceneShotImage, error) {
	var image novel.SceneShotImage
	filter := bson.M{
		"chapter_id":   chapterID,
		"scene_number": sceneNumber,
		"shot_number":  shotNumber,
		"deleted_at":   nil,
	}
	if err := r.coll.FindOne(ctx, filter).Decode(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

// UpdateStatus 更新状态
func (r *SceneShotImageRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		}},
	)
	return err
}

// Delete 软删除
func (r *SceneShotImageRepo) Delete(ctx context.Context, id string) error {
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
