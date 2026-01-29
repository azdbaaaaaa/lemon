package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ChapterRepository 章节仓库接口（供 service 层依赖）
type ChapterRepository interface {
	Create(ctx context.Context, ch *novel.Chapter) error
	FindByID(ctx context.Context, id string) (*novel.Chapter, error)
	FindByNovelID(ctx context.Context, novelID string) ([]*novel.Chapter, error)
	UpdateNarrationText(ctx context.Context, chapterID string, narrationText string) error
}

// ChapterRepo 章节仓库
type ChapterRepo struct {
	coll *mongo.Collection
}

// NewChapterRepo 创建章节仓库
func NewChapterRepo(db *mongo.Database) *ChapterRepo {
	var c novel.Chapter
	return &ChapterRepo{coll: db.Collection(c.Collection())}
}

// Create 创建章节
func (r *ChapterRepo) Create(ctx context.Context, ch *novel.Chapter) error {
	now := time.Now()
	ch.CreatedAt = now
	ch.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, ch)
	return err
}

// FindByID 根据ID查询章节
func (r *ChapterRepo) FindByID(ctx context.Context, id string) (*novel.Chapter, error) {
	var ch novel.Chapter
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// FindByNovelID 查询某小说的章节（按sequence排序）
func (r *ChapterRepo) FindByNovelID(ctx context.Context, novelID string) ([]*novel.Chapter, error) {
	filter := bson.M{"novel_id": novelID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var chapters []*novel.Chapter
	if err := cur.All(ctx, &chapters); err != nil {
		return nil, err
	}
	return chapters, nil
}

// UpdateNarrationText 更新章节的 narration_text
func (r *ChapterRepo) UpdateNarrationText(ctx context.Context, chapterID string, narrationText string) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": chapterID},
		bson.M{"$set": bson.M{
			"narration_text": narrationText,
			"updated_at":     time.Now(),
		}},
	)
	return err
}
