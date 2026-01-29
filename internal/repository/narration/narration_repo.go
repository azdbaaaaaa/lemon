package narration

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/narration"
)

// NarrationRepository 解说文案仓库接口
type NarrationRepository interface {
	Create(ctx context.Context, n *narration.Narration) error
	FindByID(ctx context.Context, id string) (*narration.Narration, error)
	FindByChapterID(ctx context.Context, chapterID string) (*narration.Narration, error)
	UpdateContent(ctx context.Context, id string, content map[string]interface{}) error
	UpdateStatus(ctx context.Context, id string, status string) error
	Delete(ctx context.Context, id string) error
}

// NarrationRepo 解说文案仓库实现
type NarrationRepo struct {
	coll *mongo.Collection
}

// NewNarrationRepo 创建解说文案仓库
func NewNarrationRepo(db *mongo.Database) *NarrationRepo {
	var n narration.Narration
	return &NarrationRepo{coll: db.Collection(n.Collection())}
}

// Create 创建解说文案
func (r *NarrationRepo) Create(ctx context.Context, n *narration.Narration) error {
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now
	if n.Status == "" {
		n.Status = "completed" // 默认状态为已完成
	}
	_, err := r.coll.InsertOne(ctx, n)
	return err
}

// FindByID 根据ID查询解说文案
func (r *NarrationRepo) FindByID(ctx context.Context, id string) (*narration.Narration, error) {
	var n narration.Narration
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// FindByChapterID 根据章节ID查询解说文案（返回最新的未删除的）
func (r *NarrationRepo) FindByChapterID(ctx context.Context, chapterID string) (*narration.Narration, error) {
	var n narration.Narration
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	if err := r.coll.FindOne(ctx, filter, opts).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// UpdateContent 更新解说文案内容
func (r *NarrationRepo) UpdateContent(ctx context.Context, id string, content map[string]interface{}) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{
			"content":    content,
			"updated_at": time.Now(),
		}},
	)
	return err
}

// UpdateStatus 更新解说文案状态
func (r *NarrationRepo) UpdateStatus(ctx context.Context, id string, status string) error {
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

// Delete 软删除解说文案
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
