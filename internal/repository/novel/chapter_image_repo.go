package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ChapterImageRepository 章节图片仓库接口（供 service 层依赖）
type ChapterImageRepository interface {
	Create(ctx context.Context, image *novel.ChapterImage) error
	FindByID(ctx context.Context, id string) (*novel.ChapterImage, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.ChapterImage, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.ChapterImage, error)
	FindBySceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.ChapterImage, error)
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.ChapterImage, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	Delete(ctx context.Context, id string) error
}

// ChapterImageRepo 章节图片仓库
type ChapterImageRepo struct {
	coll *mongo.Collection
}

// NewChapterImageRepo 创建章节图片仓库
func NewChapterImageRepo(db *mongo.Database) *ChapterImageRepo {
	var c novel.ChapterImage
	return &ChapterImageRepo{coll: db.Collection(c.Collection())}
}

// Create 创建图片记录
func (r *ChapterImageRepo) Create(ctx context.Context, image *novel.ChapterImage) error {
	now := time.Now()
	image.CreatedAt = now
	image.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, image)
	return err
}

// FindByID 根据ID查询
func (r *ChapterImageRepo) FindByID(ctx context.Context, id string) (*novel.ChapterImage, error) {
	var image novel.ChapterImage
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

// FindByChapterID 查询章节的所有图片
func (r *ChapterImageRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.ChapterImage, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.ChapterImage
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByNarrationID 查询章节解说的所有图片
func (r *ChapterImageRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.ChapterImage, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.ChapterImage
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindBySceneAndShot 根据场景和特写编号查询
func (r *ChapterImageRepo) FindBySceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.ChapterImage, error) {
	var image novel.ChapterImage
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
func (r *ChapterImageRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.ChapterImage, error) {
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.ChapterImage
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindVersionsByChapterID 查询章节的所有图片版本号
func (r *ChapterImageRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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
func (r *ChapterImageRepo) UpdateStatus(ctx context.Context, id string, status string) error {
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
func (r *ChapterImageRepo) Delete(ctx context.Context, id string) error {
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
