package workflow

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/workflow"
)

// WorkflowRepository 工作流仓库接口
type WorkflowRepository interface {
	Create(ctx context.Context, w *workflow.Workflow) error
	FindByID(ctx context.Context, id, userID string) (*workflow.Workflow, error)
	ListByUser(ctx context.Context, userID string, page, pageSize int64, status, stage string) ([]*workflow.Workflow, int64, error)
	Update(ctx context.Context, w *workflow.Workflow) error
}

// Repo 实现 WorkflowRepository
type Repo struct {
	coll *mongo.Collection
}

// NewRepo 创建工作流仓库
func NewRepo(db *mongo.Database) *Repo {
	var w workflow.Workflow
	return &Repo{coll: db.Collection(w.Collection())}
}

// Create 创建工作流
func (r *Repo) Create(ctx context.Context, w *workflow.Workflow) error {
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, w)
	return err
}

// FindByID 根据ID和用户ID查询工作流（确保归属）
func (r *Repo) FindByID(ctx context.Context, id, userID string) (*workflow.Workflow, error) {
	var w workflow.Workflow
	filter := bson.M{"id": id}
	if userID != "" {
		filter["user_id"] = userID
	}
	if err := r.coll.FindOne(ctx, filter).Decode(&w); err != nil {
		return nil, err
	}
	return &w, nil
}

// ListByUser 查询用户的工作流列表（支持状态/阶段筛选 + 分页）
func (r *Repo) ListByUser(ctx context.Context, userID string, page, pageSize int64, status, stage string) ([]*workflow.Workflow, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	if status != "" {
		filter["status"] = status
	}
	if stage != "" {
		filter["current_stage"] = stage
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)

	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var list []*workflow.Workflow
	if err := cur.All(ctx, &list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Update 更新工作流
func (r *Repo) Update(ctx context.Context, w *workflow.Workflow) error {
	w.UpdatedAt = time.Now()
	filter := bson.M{"id": w.ID}
	update := bson.M{"$set": w}
	_, err := r.coll.UpdateOne(ctx, filter, update)
	return err
}

