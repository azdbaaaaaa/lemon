package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ImageRepository 图片仓库接口（供 service 层依赖）
type ImageRepository interface {
	Create(ctx context.Context, image *novel.Image) error
	FindByID(ctx context.Context, id string) (*novel.Image, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Image, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Image, error)
	FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Image, error)
	FindBySceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.Image, error)
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.Image, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error
	Delete(ctx context.Context, id string) error
}

// ImageRepo 图片仓库
type ImageRepo struct {
	coll *mongo.Collection
}

// NewImageRepo 创建图片仓库
func NewImageRepo(db *mongo.Database) *ImageRepo {
	var i novel.Image
	return &ImageRepo{coll: db.Collection(i.Collection())}
}

// Create 创建图片记录
func (r *ImageRepo) Create(ctx context.Context, image *novel.Image) error {
	now := time.Now()
	image.CreatedAt = now
	image.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, image)
	return err
}

// FindByID 根据ID查询
func (r *ImageRepo) FindByID(ctx context.Context, id string) (*novel.Image, error) {
	var image novel.Image
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

// FindByChapterID 查询章节的所有图片
func (r *ImageRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Image, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByNarrationID 查询解说的所有图片
func (r *ImageRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Image, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByNarrationIDAndVersion 根据解说ID和版本号查询所有图片（按 sequence 排序）
func (r *ImageRepo) FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Image, error) {
	filter := bson.M{"narration_id": narrationID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindBySceneAndShot 根据场景和镜头编号查询
func (r *ImageRepo) FindBySceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.Image, error) {
	var image novel.Image
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

// FindByChapterIDAndVersion 根据章节ID和版本号查询图片
func (r *ImageRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.Image, error) {
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindVersionsByChapterID 查询章节的所有图片版本号
func (r *ImageRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetProjection(bson.M{"version": 1}).SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var versions []int
	versionSet := make(map[int]bool)
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		if version, ok := doc["version"].(int32); ok && version > 0 {
			v := int(version)
			if !versionSet[v] {
				versions = append(versions, v)
				versionSet[v] = true
			}
		} else if version, ok := doc["version"].(int); ok && version > 0 {
			if !versionSet[version] {
				versions = append(versions, version)
				versionSet[version] = true
			}
		}
	}
	return versions, nil
}

// UpdateStatus 更新状态
func (r *ImageRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error {
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
func (r *ImageRepo) Delete(ctx context.Context, id string) error {
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
