package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ChapterAudioRepository 章节音频仓库接口
type ChapterAudioRepository interface {
	Create(ctx context.Context, a *novel.ChapterAudio) error
	FindByID(ctx context.Context, id string) (*novel.ChapterAudio, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.ChapterAudio, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.ChapterAudio, error)
	FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.ChapterAudio, error)
	FindVersionsByNarrationID(ctx context.Context, narrationID string) ([]int, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateVersion(ctx context.Context, id string, version int) error
	Delete(ctx context.Context, id string) error
}

// ChapterAudioRepo 章节音频仓库实现
type ChapterAudioRepo struct {
	coll *mongo.Collection
}

// NewChapterAudioRepo 创建章节音频仓库
func NewChapterAudioRepo(db *mongo.Database) *ChapterAudioRepo {
	var c novel.ChapterAudio
	return &ChapterAudioRepo{coll: db.Collection(c.Collection())}
}

// Create 创建音频记录
func (r *ChapterAudioRepo) Create(ctx context.Context, a *novel.ChapterAudio) error {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Status == "" {
		a.Status = "pending" // 默认状态为待处理
	}
	if a.Version == 0 {
		a.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, a)
	return err
}

// FindByID 根据ID查询音频
func (r *ChapterAudioRepo) FindByID(ctx context.Context, id string) (*novel.ChapterAudio, error) {
	var a novel.ChapterAudio
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// FindByNarrationID 根据章节解说ID查询所有音频
func (r *ChapterAudioRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.ChapterAudio, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var audios []*novel.ChapterAudio
	if err := cursor.All(ctx, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// FindByChapterID 根据章节ID查询所有音频
func (r *ChapterAudioRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.ChapterAudio, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var audios []*novel.ChapterAudio
	if err := cursor.All(ctx, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// FindByNarrationIDAndVersion 根据章节解说ID和版本号查询所有音频
func (r *ChapterAudioRepo) FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.ChapterAudio, error) {
	filter := bson.M{"narration_id": narrationID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var audios []*novel.ChapterAudio
	if err := cursor.All(ctx, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// FindVersionsByNarrationID 查询章节解说的所有版本号
func (r *ChapterAudioRepo) FindVersionsByNarrationID(ctx context.Context, narrationID string) ([]int, error) {
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
func (r *ChapterAudioRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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

// UpdateStatus 更新音频状态
func (r *ChapterAudioRepo) UpdateStatus(ctx context.Context, id string, status string) error {
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

// UpdateVersion 更新章节音频版本号
func (r *ChapterAudioRepo) UpdateVersion(ctx context.Context, id string, version int) error {
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

// Delete 软删除音频
func (r *ChapterAudioRepo) Delete(ctx context.Context, id string) error {
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
