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
	FindByChapterID(ctx context.Context, chapterID string) (*novel.Subtitle, error)
	FindByNarrationID(ctx context.Context, narrationID string) (*novel.Subtitle, error)
	UpdateStatus(ctx context.Context, id string, status string) error
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
	if s.Status == "" {
		s.Status = "pending" // 默认状态为待处理
	}
	if s.Format == "" {
		s.Format = "ass" // 默认格式为 ASS
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

// FindByChapterID 根据章节ID查询字幕（返回最新的未删除的）
func (r *SubtitleRepo) FindByChapterID(ctx context.Context, chapterID string) (*novel.Subtitle, error) {
	var s novel.Subtitle
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByNarrationID 根据解说文案ID查询字幕（返回最新的未删除的）
func (r *SubtitleRepo) FindByNarrationID(ctx context.Context, narrationID string) (*novel.Subtitle, error) {
	var s novel.Subtitle
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateStatus 更新字幕状态
func (r *SubtitleRepo) UpdateStatus(ctx context.Context, id string, status string) error {
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
