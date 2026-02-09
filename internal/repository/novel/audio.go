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
	FindByShotID(ctx context.Context, shotID string) ([]*novel.Audio, error)                                                            // 查询分镜音频
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Audio, error)                                                      // 查询章节的所有音频
	FindByNovelIDAndType(ctx context.Context, novelID string, audioType novel.AudioType) ([]*novel.Audio, error)                        // 查询完整音频
	FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, audioType novel.AudioType, version int) ([]*novel.Audio, error)   // 查询分镜音频（指定版本）
	FindByNovelIDAndTypeAndVersion(ctx context.Context, novelID string, audioType novel.AudioType, version int) ([]*novel.Audio, error) // 查询完整音频（指定版本）
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error
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

// FindByShotID 查询分镜音频（按创建时间排序）
func (r *AudioRepo) FindByShotID(ctx context.Context, shotID string) ([]*novel.Audio, error) {
	filter := bson.M{"shot_id": shotID, "audio_type": novel.AudioTypeShot, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByChapterID 查询章节的所有音频（按创建时间排序）
func (r *AudioRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Audio, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByNovelIDAndType 查询完整音频（按创建时间排序）
func (r *AudioRepo) FindByNovelIDAndType(ctx context.Context, novelID string, audioType novel.AudioType) ([]*novel.Audio, error) {
	filter := bson.M{"novel_id": novelID, "audio_type": audioType, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByShotIDAndTypeAndVersion 查询分镜音频（指定版本）
func (r *AudioRepo) FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, audioType novel.AudioType, version int) ([]*novel.Audio, error) {
	filter := bson.M{
		"shot_id":    shotID,
		"audio_type": audioType,
		"version":    version,
		"deleted_at": nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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

// FindByNovelIDAndTypeAndVersion 查询完整音频（指定版本）
func (r *AudioRepo) FindByNovelIDAndTypeAndVersion(ctx context.Context, novelID string, audioType novel.AudioType, version int) ([]*novel.Audio, error) {
	filter := bson.M{
		"novel_id":   novelID,
		"audio_type": audioType,
		"version":    version,
		"deleted_at": nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
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
