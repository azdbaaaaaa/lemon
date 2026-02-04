package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// VideoRepository 视频仓库接口
type VideoRepository interface {
	Create(ctx context.Context, v *novel.Video) error
	FindByID(ctx context.Context, id string) (*novel.Video, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Video, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Video, error)
	FindByChapterIDAndType(ctx context.Context, chapterID string, videoType string) ([]*novel.Video, error)
	FindByStatus(ctx context.Context, status string) ([]*novel.Video, error) // 用于轮询
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.Video, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status string, errorMsg string) error
	UpdateVideoResourceID(ctx context.Context, id string, resourceID string, duration float64, prompt string) error
	UpdateVersion(ctx context.Context, id string, version int) error
	Delete(ctx context.Context, id string) error
}

// VideoRepo 视频仓库实现
type VideoRepo struct {
	coll *mongo.Collection
}

// NewVideoRepo 创建视频仓库
func NewVideoRepo(db *mongo.Database) *VideoRepo {
	var v novel.Video
	return &VideoRepo{coll: db.Collection(v.Collection())}
}

// Create 创建视频记录
func (r *VideoRepo) Create(ctx context.Context, v *novel.Video) error {
	now := time.Now()
	v.CreatedAt = now
	v.UpdatedAt = now
	if v.Status == "" {
		v.Status = "pending" // 默认状态为待处理
	}
	if v.Version == 0 {
		v.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, v)
	return err
}

// FindByID 根据ID查询视频
func (r *VideoRepo) FindByID(ctx context.Context, id string) (*novel.Video, error) {
	var v novel.Video
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// FindByChapterID 根据章节ID查询所有视频
func (r *VideoRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Video, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var videos []*novel.Video
	if err := cursor.All(ctx, &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// FindByNarrationID 根据解说ID查询所有视频
func (r *VideoRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Video, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var videos []*novel.Video
	if err := cursor.All(ctx, &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// FindByChapterIDAndType 根据章节ID和视频类型查询视频
func (r *VideoRepo) FindByChapterIDAndType(ctx context.Context, chapterID string, videoType string) ([]*novel.Video, error) {
	filter := bson.M{"chapter_id": chapterID, "video_type": videoType, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var videos []*novel.Video
	if err := cursor.All(ctx, &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// FindByStatus 根据状态查询视频（用于轮询）
func (r *VideoRepo) FindByStatus(ctx context.Context, status string) ([]*novel.Video, error) {
	filter := bson.M{"status": status, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var videos []*novel.Video
	if err := cursor.All(ctx, &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// FindByChapterIDAndVersion 根据章节ID和版本号查询视频
func (r *VideoRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) ([]*novel.Video, error) {
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var videos []*novel.Video
	if err := cursor.All(ctx, &videos); err != nil {
		return nil, err
	}
	return videos, nil
}

// FindVersionsByChapterID 查询章节的所有视频版本号
func (r *VideoRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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

// UpdateStatus 更新视频状态
func (r *VideoRepo) UpdateStatus(ctx context.Context, id string, status string, errorMsg string) error {
	update := bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errorMsg != "" {
		update["error_message"] = errorMsg
	}
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": update},
	)
	return err
}

// UpdateVideoResourceID 更新视频资源ID和相关信息
func (r *VideoRepo) UpdateVideoResourceID(ctx context.Context, id string, resourceID string, duration float64, prompt string) error {
	update := bson.M{
		"video_resource_id": resourceID,
		"duration":          duration,
		"updated_at":        time.Now(),
	}
	if prompt != "" {
		update["prompt"] = prompt
	}
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": update},
	)
	return err
}

// UpdateVersion 更新视频版本号
func (r *VideoRepo) UpdateVersion(ctx context.Context, id string, version int) error {
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

// Delete 软删除视频
func (r *VideoRepo) Delete(ctx context.Context, id string) error {
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
