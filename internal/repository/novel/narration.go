package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// NarrationRepository 解说仓库接口
type NarrationRepository interface {
	Create(ctx context.Context, n *novel.Narration) error
	FindByID(ctx context.Context, id string) (*novel.Narration, error)
	FindByChapterID(ctx context.Context, chapterID string) (*novel.Narration, error)
	FindAllByChapterID(ctx context.Context, chapterID string) ([]*novel.Narration, error)
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) (*novel.Narration, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error
	UpdateVersion(ctx context.Context, id string, version int) error
	Delete(ctx context.Context, id string) error
}

// NarrationRepo 解说仓库实现
type NarrationRepo struct {
	coll *mongo.Collection
}

// NewNarrationRepo 创建解说仓库
func NewNarrationRepo(db *mongo.Database) *NarrationRepo {
	var n novel.Narration
	return &NarrationRepo{coll: db.Collection(n.Collection())}
}

// Create 创建解说
func (r *NarrationRepo) Create(ctx context.Context, n *novel.Narration) error {
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now
	if n.Status == "" || n.Status == novel.TaskStatus("") {
		n.Status = novel.TaskStatusCompleted // 默认状态为已完成
	}
	if n.Version == 0 {
		n.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, n)
	return err
}

// FindByID 根据ID查询解说
func (r *NarrationRepo) FindByID(ctx context.Context, id string) (*novel.Narration, error) {
	var n novel.Narration
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindByChapterID 根据章节ID查询解说（返回最新的未删除的）
func (r *NarrationRepo) FindByChapterID(ctx context.Context, chapterID string) (*novel.Narration, error) {
	var n novel.Narration
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindAllByChapterID 根据章节ID查询所有解说（按 version desc, created_at desc 排序）
func (r *NarrationRepo) FindAllByChapterID(ctx context.Context, chapterID string) ([]*novel.Narration, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"version": -1, "created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var narrations []*novel.Narration
	if err := cur.All(ctx, &narrations); err != nil {
		return nil, err
	}
	return narrations, nil
}

// FindByChapterIDAndVersion 根据章节ID和版本号查询解说
func (r *NarrationRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) (*novel.Narration, error) {
	var n novel.Narration
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindVersionsByChapterID 查询章节的所有版本号
func (r *NarrationRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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

// UpdateStatus 更新解说状态
func (r *NarrationRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error {
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

// UpdateVersion 更新解说版本号
func (r *NarrationRepo) UpdateVersion(ctx context.Context, id string, version int) error {
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

// Delete 软删除解说
func (r *NarrationRepo) Delete(ctx context.Context, id string) error {
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
