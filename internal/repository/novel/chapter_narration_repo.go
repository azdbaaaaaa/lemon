package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ChapterNarrationRepository 章节解说仓库接口
type ChapterNarrationRepository interface {
	Create(ctx context.Context, n *novel.ChapterNarration) error
	FindByID(ctx context.Context, id string) (*novel.ChapterNarration, error)
	FindByChapterID(ctx context.Context, chapterID string) (*novel.ChapterNarration, error)
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) (*novel.ChapterNarration, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateContent(ctx context.Context, id string, content *novel.NarrationContent) error
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateVersion(ctx context.Context, id string, version int) error
	Delete(ctx context.Context, id string) error
}

// ChapterNarrationRepo 章节解说仓库实现
type ChapterNarrationRepo struct {
	coll *mongo.Collection
}

// NewChapterNarrationRepo 创建章节解说仓库
func NewChapterNarrationRepo(db *mongo.Database) *ChapterNarrationRepo {
	var c novel.ChapterNarration
	return &ChapterNarrationRepo{coll: db.Collection(c.Collection())}
}

// Create 创建章节解说
func (r *ChapterNarrationRepo) Create(ctx context.Context, n *novel.ChapterNarration) error {
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now
	if n.Status == "" {
		n.Status = "completed" // 默认状态为已完成
	}
	if n.Version == 0 {
		n.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, n)
	return err
}

// FindByID 根据ID查询章节解说
func (r *ChapterNarrationRepo) FindByID(ctx context.Context, id string) (*novel.ChapterNarration, error) {
	var n novel.ChapterNarration
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindByChapterID 根据章节ID查询章节解说（返回最新的未删除的）
func (r *ChapterNarrationRepo) FindByChapterID(ctx context.Context, chapterID string) (*novel.ChapterNarration, error) {
	var n novel.ChapterNarration
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindByChapterIDAndVersion 根据章节ID和版本号查询章节解说
func (r *ChapterNarrationRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) (*novel.ChapterNarration, error) {
	var n novel.ChapterNarration
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindVersionsByChapterID 查询章节的所有版本号
func (r *ChapterNarrationRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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

// UpdateContent 更新章节解说内容
func (r *ChapterNarrationRepo) UpdateContent(ctx context.Context, id string, content *novel.NarrationContent) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{
			"content":    content,
			"updated_at": time.Now(),
		}},
	)
	return err
}

// UpdateStatus 更新章节解说状态
func (r *ChapterNarrationRepo) UpdateStatus(ctx context.Context, id string, status string) error {
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

// UpdateVersion 更新章节解说版本号
func (r *ChapterNarrationRepo) UpdateVersion(ctx context.Context, id string, version int) error {
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

// Delete 软删除章节解说
func (r *ChapterNarrationRepo) Delete(ctx context.Context, id string) error {
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
