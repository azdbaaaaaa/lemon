package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// NovelRepository 小说仓库接口（供 service 层依赖）
type NovelRepository interface {
	Create(ctx context.Context, novel *novel.Novel) error
	FindByID(ctx context.Context, id string) (*novel.Novel, error)
	FindByWorkflowID(ctx context.Context, workflowID string, limit int64) ([]*novel.Novel, error)
}

// NovelRepo 小说仓库
type NovelRepo struct {
	coll *mongo.Collection
}

// NewNovelRepo 创建小说仓库
func NewNovelRepo(db *mongo.Database) *NovelRepo {
	var n novel.Novel
	return &NovelRepo{coll: db.Collection(n.Collection())}
}

// Create 创建小说
func (r *NovelRepo) Create(ctx context.Context, novel *novel.Novel) error {
	now := time.Now()
	novel.CreatedAt = now
	novel.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, novel)
	return err
}

// FindByID 根据ID查询
func (r *NovelRepo) FindByID(ctx context.Context, id string) (*novel.Novel, error) {
	var n novel.Novel
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindByWorkflowID 根据workflow_id查询（按创建时间倒序）
func (r *NovelRepo) FindByWorkflowID(ctx context.Context, workflowID string, limit int64) ([]*novel.Novel, error) {
	filter := bson.M{"workflow_id": workflowID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var novels []*novel.Novel
	if err := cur.All(ctx, &novels); err != nil {
		return nil, err
	}
	return novels, nil
}
