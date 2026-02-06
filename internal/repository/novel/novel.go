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
	ListByUser(ctx context.Context, userID string, page, pageSize int64) ([]*novel.Novel, int64, error)
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

// ListByUser 根据用户ID查询小说列表（分页）
func (r *NovelRepo) ListByUser(ctx context.Context, userID string, page, pageSize int64) ([]*novel.Novel, int64, error) {
	filter := bson.M{"user_id": userID, "deleted_at": nil}
	
	// 计算总数
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)
	
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var novels []*novel.Novel
	if err := cur.All(ctx, &novels); err != nil {
		return nil, 0, err
	}
	return novels, total, nil
}
