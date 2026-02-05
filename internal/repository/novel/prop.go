package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// PropRepository 道具仓库接口
type PropRepository interface {
	Create(ctx context.Context, prop *novel.Prop) error
	CreateMany(ctx context.Context, props []*novel.Prop) error
	FindByID(ctx context.Context, id string) (*novel.Prop, error)
	FindByNovelID(ctx context.Context, novelID string) ([]*novel.Prop, error)
	FindByName(ctx context.Context, novelID, name string) (*novel.Prop, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error
	Delete(ctx context.Context, id string) error
}

// PropRepo 道具仓库实现
type PropRepo struct {
	coll *mongo.Collection
}

// NewPropRepo 创建道具仓库
func NewPropRepo(db *mongo.Database) *PropRepo {
	var p novel.Prop
	return &PropRepo{coll: db.Collection(p.Collection())}
}

// Create 创建道具
func (r *PropRepo) Create(ctx context.Context, prop *novel.Prop) error {
	now := time.Now()
	prop.CreatedAt = now
	prop.UpdatedAt = now
	if prop.Status == "" || prop.Status == novel.TaskStatus("") {
		prop.Status = novel.TaskStatusPending
	}
	_, err := r.coll.InsertOne(ctx, prop)
	return err
}

// CreateMany 批量创建道具
func (r *PropRepo) CreateMany(ctx context.Context, props []*novel.Prop) error {
	if len(props) == 0 {
		return nil
	}
	now := time.Now()
	docs := make([]interface{}, len(props))
	for i, prop := range props {
		prop.CreatedAt = now
		prop.UpdatedAt = now
		if prop.Status == "" {
			prop.Status = novel.TaskStatusPending
		}
		docs[i] = prop
	}
	_, err := r.coll.InsertMany(ctx, docs)
	return err
}

// FindByID 根据ID查询道具
func (r *PropRepo) FindByID(ctx context.Context, id string) (*novel.Prop, error) {
	var prop novel.Prop
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&prop); err != nil {
		return nil, err
	}
	return &prop, nil
}

// FindByNovelID 根据小说ID查询所有道具
func (r *PropRepo) FindByNovelID(ctx context.Context, novelID string) ([]*novel.Prop, error) {
	filter := bson.M{"novel_id": novelID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var props []*novel.Prop
	if err := cur.All(ctx, &props); err != nil {
		return nil, err
	}
	return props, nil
}

// FindByName 根据小说ID和名称查询道具
func (r *PropRepo) FindByName(ctx context.Context, novelID, name string) (*novel.Prop, error) {
	var prop novel.Prop
	filter := bson.M{"novel_id": novelID, "name": name, "deleted_at": nil}
	if err := r.coll.FindOne(ctx, filter).Decode(&prop); err != nil {
		return nil, err
	}
	return &prop, nil
}

// Update 更新道具
func (r *PropRepo) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": updates},
	)
	return err
}

// UpdateStatus 更新道具状态
func (r *PropRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error {
	update := bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errorMessage != "" {
		update["error_message"] = errorMessage
	} else {
		update["$unset"] = bson.M{"error_message": ""}
	}
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": update},
	)
	return err
}

// Delete 软删除道具
func (r *PropRepo) Delete(ctx context.Context, id string) error {
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
