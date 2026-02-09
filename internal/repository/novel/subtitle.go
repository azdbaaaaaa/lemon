package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// SubtitleRepository 字幕仓库接口
type SubtitleRepository interface {
	Create(ctx context.Context, s *novel.Subtitle) error
	FindByID(ctx context.Context, id string) (*novel.Subtitle, error)
	FindByShotID(ctx context.Context, shotID string) ([]*novel.Subtitle, error)                                                                  // 查询分镜字幕
	FindByNovelIDAndType(ctx context.Context, novelID string, subtitleType novel.SubtitleType) ([]*novel.Subtitle, error)                        // 查询完整字幕
	FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, subtitleType novel.SubtitleType, version int) ([]*novel.Subtitle, error)   // 查询分镜字幕（指定版本）
	FindByNovelIDAndTypeAndVersion(ctx context.Context, novelID string, subtitleType novel.SubtitleType, version int) ([]*novel.Subtitle, error) // 查询完整字幕（指定版本）
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error
	Delete(ctx context.Context, id string) error
}

// SubtitleRepo 字幕仓库实现
type SubtitleRepo struct {
	coll *mongo.Collection
}

// NewSubtitleRepo 创建字幕仓库
func NewSubtitleRepo(db *mongo.Database) *SubtitleRepo {
	var s novel.Subtitle
	return &SubtitleRepo{coll: db.Collection(s.Collection())}
}

// Create 创建字幕记录
func (r *SubtitleRepo) Create(ctx context.Context, s *novel.Subtitle) error {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.Status == "" || s.Status == novel.TaskStatus("") {
		s.Status = novel.TaskStatusPending // 默认状态为待处理
	}
	if s.Format == "" || s.Format == novel.SubtitleFormat("") {
		s.Format = novel.SubtitleFormatASS // 默认格式为 ASS
	}
	if s.Version == 0 {
		s.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

// FindByID 根据ID查询字幕
func (r *SubtitleRepo) FindByID(ctx context.Context, id string) (*novel.Subtitle, error) {
	var s novel.Subtitle
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByShotID 查询分镜字幕（按创建时间排序）
func (r *SubtitleRepo) FindByShotID(ctx context.Context, shotID string) ([]*novel.Subtitle, error) {
	filter := bson.M{"shot_id": shotID, "subtitle_type": novel.SubtitleTypeShot, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var subtitles []*novel.Subtitle
	if err := cur.All(ctx, &subtitles); err != nil {
		return nil, err
	}
	return subtitles, nil
}

// FindByNovelIDAndType 查询完整字幕（按创建时间排序）
func (r *SubtitleRepo) FindByNovelIDAndType(ctx context.Context, novelID string, subtitleType novel.SubtitleType) ([]*novel.Subtitle, error) {
	filter := bson.M{"novel_id": novelID, "subtitle_type": subtitleType, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var subtitles []*novel.Subtitle
	if err := cur.All(ctx, &subtitles); err != nil {
		return nil, err
	}
	return subtitles, nil
}

// FindByShotIDAndTypeAndVersion 查询分镜字幕（指定版本）
func (r *SubtitleRepo) FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, subtitleType novel.SubtitleType, version int) ([]*novel.Subtitle, error) {
	filter := bson.M{
		"shot_id":       shotID,
		"subtitle_type": subtitleType,
		"version":       version,
		"deleted_at":    nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var subtitles []*novel.Subtitle
	if err := cur.All(ctx, &subtitles); err != nil {
		return nil, err
	}
	return subtitles, nil
}

// FindByNovelIDAndTypeAndVersion 查询完整字幕（指定版本）
func (r *SubtitleRepo) FindByNovelIDAndTypeAndVersion(ctx context.Context, novelID string, subtitleType novel.SubtitleType, version int) ([]*novel.Subtitle, error) {
	filter := bson.M{
		"novel_id":      novelID,
		"subtitle_type": subtitleType,
		"version":       version,
		"deleted_at":    nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var subtitles []*novel.Subtitle
	if err := cur.All(ctx, &subtitles); err != nil {
		return nil, err
	}
	return subtitles, nil
}

// UpdateStatus 更新字幕状态
func (r *SubtitleRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error {
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

// Delete 软删除字幕
func (r *SubtitleRepo) Delete(ctx context.Context, id string) error {
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
