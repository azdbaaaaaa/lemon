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
	FindByShotID(ctx context.Context, shotID string) ([]*novel.Video, error)                                                            // 查询分镜视频
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Video, error)                                                      // 查询章节的所有视频
	FindByNovelIDAndType(ctx context.Context, novelID string, videoType novel.VideoType) ([]*novel.Video, error)                        // 查询完整视频
	FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, videoType novel.VideoType, version int) ([]*novel.Video, error)   // 查询分镜视频（指定版本）
	FindByNovelIDAndTypeAndVersion(ctx context.Context, novelID string, videoType novel.VideoType, version int) ([]*novel.Video, error) // 查询完整视频（指定版本）
	FindByStatus(ctx context.Context, status novel.VideoStatus) ([]*novel.Video, error)                                                 // 用于轮询
	UpdateStatus(ctx context.Context, id string, status novel.VideoStatus, errorMsg string) error
	UpdateVideoResourceID(ctx context.Context, id string, resourceID string, duration float64, prompt string) error
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
	if v.Status == "" || v.Status == novel.VideoStatus("") {
		v.Status = novel.VideoStatusPending // 默认状态为待处理
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

// FindByShotID 查询分镜视频（按创建时间排序）
func (r *VideoRepo) FindByShotID(ctx context.Context, shotID string) ([]*novel.Video, error) {
	filter := bson.M{"shot_id": shotID, "video_type": novel.VideoTypeShot, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByChapterID 查询章节的所有视频（按创建时间排序）
func (r *VideoRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Video, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByNovelIDAndType 查询完整视频（按创建时间排序）
func (r *VideoRepo) FindByNovelIDAndType(ctx context.Context, novelID string, videoType novel.VideoType) ([]*novel.Video, error) {
	filter := bson.M{"novel_id": novelID, "video_type": videoType, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByShotIDAndTypeAndVersion 查询分镜视频（指定版本）
func (r *VideoRepo) FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, videoType novel.VideoType, version int) ([]*novel.Video, error) {
	filter := bson.M{
		"shot_id":    shotID,
		"video_type": videoType,
		"version":    version,
		"deleted_at": nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByNovelIDAndTypeAndVersion 查询完整视频（指定版本）
func (r *VideoRepo) FindByNovelIDAndTypeAndVersion(ctx context.Context, novelID string, videoType novel.VideoType, version int) ([]*novel.Video, error) {
	filter := bson.M{
		"novel_id":   novelID,
		"video_type": videoType,
		"version":    version,
		"deleted_at": nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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
func (r *VideoRepo) FindByStatus(ctx context.Context, status novel.VideoStatus) ([]*novel.Video, error) {
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

// UpdateStatus 更新视频状态
func (r *VideoRepo) UpdateStatus(ctx context.Context, id string, status novel.VideoStatus, errorMsg string) error {
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
