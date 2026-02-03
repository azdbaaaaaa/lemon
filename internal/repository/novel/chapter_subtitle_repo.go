package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ChapterSubtitleRepository 章节字幕仓库接口
type ChapterSubtitleRepository interface {
	Create(ctx context.Context, s *novel.ChapterSubtitle) error
	FindByID(ctx context.Context, id string) (*novel.ChapterSubtitle, error)
	FindByChapterID(ctx context.Context, chapterID string) (*novel.ChapterSubtitle, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.ChapterSubtitle, error)
	FindByNarrationIDAndSequence(ctx context.Context, narrationID string, sequence int) (*novel.ChapterSubtitle, error)
	FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) (*novel.ChapterSubtitle, error)
	FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateVersion(ctx context.Context, id string, version int) error
	Delete(ctx context.Context, id string) error
}

// ChapterSubtitleRepo 章节字幕仓库实现
type ChapterSubtitleRepo struct {
	coll *mongo.Collection
}

// NewChapterSubtitleRepo 创建章节字幕仓库
func NewChapterSubtitleRepo(db *mongo.Database) *ChapterSubtitleRepo {
	var c novel.ChapterSubtitle
	return &ChapterSubtitleRepo{coll: db.Collection(c.Collection())}
}

// Create 创建字幕记录
func (r *ChapterSubtitleRepo) Create(ctx context.Context, s *novel.ChapterSubtitle) error {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.Status == "" {
		s.Status = "pending" // 默认状态为待处理
	}
	if s.Format == "" {
		s.Format = "ass" // 默认格式为 ASS
	}
	if s.Version == 0 {
		s.Version = 1 // 默认版本为 1
	}
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

// FindByID 根据ID查询字幕
func (r *ChapterSubtitleRepo) FindByID(ctx context.Context, id string) (*novel.ChapterSubtitle, error) {
	var s novel.ChapterSubtitle
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByChapterID 根据章节ID查询字幕（返回最新的未删除的）
func (r *ChapterSubtitleRepo) FindByChapterID(ctx context.Context, chapterID string) (*novel.ChapterSubtitle, error) {
	var s novel.ChapterSubtitle
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByNarrationID 根据章节解说ID查询所有字幕（按 sequence 排序）
func (r *ChapterSubtitleRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.ChapterSubtitle, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var subtitles []*novel.ChapterSubtitle
	for cur.Next(ctx) {
		var s novel.ChapterSubtitle
		if err := cur.Decode(&s); err != nil {
			continue
		}
		subtitles = append(subtitles, &s)
	}
	return subtitles, nil
}

// FindByNarrationIDAndSequence 根据章节解说ID和序号查询字幕
func (r *ChapterSubtitleRepo) FindByNarrationIDAndSequence(ctx context.Context, narrationID string, sequence int) (*novel.ChapterSubtitle, error) {
	var s novel.ChapterSubtitle
	filter := bson.M{"narration_id": narrationID, "sequence": sequence, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByChapterIDAndVersion 根据章节ID和版本号查询字幕
func (r *ChapterSubtitleRepo) FindByChapterIDAndVersion(ctx context.Context, chapterID string, version int) (*novel.ChapterSubtitle, error) {
	var s novel.ChapterSubtitle
	filter := bson.M{"chapter_id": chapterID, "version": version, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindVersionsByChapterID 查询章节的所有版本号
func (r *ChapterSubtitleRepo) FindVersionsByChapterID(ctx context.Context, chapterID string) ([]int, error) {
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

// UpdateStatus 更新字幕状态
func (r *ChapterSubtitleRepo) UpdateStatus(ctx context.Context, id string, status string) error {
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

// UpdateVersion 更新章节字幕版本号
func (r *ChapterSubtitleRepo) UpdateVersion(ctx context.Context, id string, version int) error {
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

// Delete 软删除字幕
func (r *ChapterSubtitleRepo) Delete(ctx context.Context, id string) error {
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
