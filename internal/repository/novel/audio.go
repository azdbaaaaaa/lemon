package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// AudioRepository 音频仓库接口
type AudioRepository interface {
	Create(ctx context.Context, a *novel.Audio) error
	FindByID(ctx context.Context, id string) (*novel.Audio, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Audio, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Audio, error)
	FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Audio, error)
	FindVersionsByNarrationID(ctx context.Context, narrationID string) ([]int, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error
	UpdateVersion(ctx context.Context, id string, version int) error
	Delete(ctx context.Context, id string) error
}

// AudioRepo 音频仓库实现
type AudioRepo struct {
	coll *mongo.Collection
}

// NewAudioRepo 创建音频仓库
func NewAudioRepo(db *mongo.Database) *AudioRepo {
	var a novel.Audio
	return &AudioRepo{coll: db.Collection(a.Collection())}
}

// Create 创建音频记录
func (r *AudioRepo) Create(ctx context.Context, a *novel.Audio) error {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Status == "" || a.Status == novel.TaskStatus("") {
		a.Status = novel.TaskStatusPending // 默认状态为待处理
	}
	if a.Version == 0 {
		a.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, a)
	return err
}

// FindByID 根据ID查询
func (r *AudioRepo) FindByID(ctx context.Context, id string) (*novel.Audio, error) {
	var a novel.Audio
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// FindByNarrationID 查询解说的所有音频（按sequence排序）
func (r *AudioRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Audio, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var audios []*novel.Audio
	if err := cur.All(ctx, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// FindByChapterID 查询章节的所有音频
func (r *AudioRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Audio, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var audios []*novel.Audio
	if err := cur.All(ctx, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// FindByNarrationIDAndVersion 根据解说ID和版本号查询音频
func (r *AudioRepo) FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Audio, error) {
	filter := bson.M{"narration_id": narrationID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var audios []*novel.Audio
	if err := cur.All(ctx, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// FindVersionsByNarrationID 查询解说的所有音频版本号
func (r *AudioRepo) FindVersionsByNarrationID(ctx context.Context, narrationID string) ([]int, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
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

// FindVersionsByChapterID 查询章节的所有音频版本号
func (r *AudioRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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
func (r *AudioRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error {
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

// UpdateVersion 更新版本号
func (r *AudioRepo) UpdateVersion(ctx context.Context, id string, version int) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{
			"version":    version,
			"updated_at": time.Now(),
		}},
	)
	return err
}

// Delete 软删除
func (r *AudioRepo) Delete(ctx context.Context, id string) error {
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
