package resource

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/resource"
)

// ResourceRepo 资源仓库
type ResourceRepo struct {
	collection *mongo.Collection
}

// NewResourceRepo 创建资源仓库
func NewResourceRepo(db *mongo.Database) *ResourceRepo {
	var res resource.Resource
	return &ResourceRepo{
		collection: db.Collection(res.Collection()),
	}
}

// Create 创建资源
func (r *ResourceRepo) Create(ctx context.Context, res *resource.Resource) error {
	now := time.Now()
	res.CreatedAt = now
	res.UpdatedAt = now
	if res.UploadedAt.IsZero() {
		res.UploadedAt = now
	}

	_, err := r.collection.InsertOne(ctx, res)
	return err
}

// FindByID 根据ID查询
func (r *ResourceRepo) FindByID(ctx context.Context, id string) (*resource.Resource, error) {
	var res resource.Resource
	err := r.collection.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// FindByUserID 根据用户ID查询资源列表
func (r *ResourceRepo) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*resource.Resource, int64, error) {
	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil,
	}

	// 查询总数
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	opts := options.Find().
		SetSort(bson.D{bson.E{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var resources []*resource.Resource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, 0, err
	}

	return resources, total, nil
}

// FindAll 查询所有资源列表（不限制用户ID，用于系统内部请求）
func (r *ResourceRepo) FindAll(ctx context.Context, limit, offset int) ([]*resource.Resource, int64, error) {
	filter := bson.M{
		"deleted_at": nil,
	}

	// 查询总数
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	opts := options.Find().
		SetSort(bson.D{bson.E{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var resources []*resource.Resource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, 0, err
	}

	return resources, total, nil
}

// FindByMD5 根据MD5查询（去重）
func (r *ResourceRepo) FindByMD5(ctx context.Context, md5 string) (*resource.Resource, error) {
	var res resource.Resource
	err := r.collection.FindOne(ctx, bson.M{"md5": md5, "deleted_at": nil}).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// FindByStorageKey 根据存储路径查询
func (r *ResourceRepo) FindByStorageKey(ctx context.Context, storageKey string) (*resource.Resource, error) {
	var res resource.Resource
	err := r.collection.FindOne(ctx, bson.M{"storage_key": storageKey, "deleted_at": nil}).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Update 更新资源
func (r *ResourceRepo) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": updates},
	)
	return err
}

// Delete 删除资源（软删除）
func (r *ResourceRepo) Delete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{
			"$set": bson.M{
				"deleted_at": now,
				"status":     resource.ResourceStatusDeleted,
				"updated_at": now,
			},
		},
	)
	return err
}

// CreateUploadSession 创建上传会话
func (r *ResourceRepo) CreateUploadSession(ctx context.Context, session *resource.UploadSession) error {
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err := r.collection.Database().Collection(session.Collection()).InsertOne(ctx, session)
	return err
}

// FindUploadSession 查询上传会话
func (r *ResourceRepo) FindUploadSession(ctx context.Context, sessionID string) (*resource.UploadSession, error) {
	var session resource.UploadSession
	err := r.collection.Database().Collection(session.Collection()).FindOne(ctx, bson.M{"id": sessionID}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateUploadSession 更新上传会话
func (r *ResourceRepo) UpdateUploadSession(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	var session resource.UploadSession
	_, err := r.collection.Database().Collection(session.Collection()).UpdateOne(
		ctx,
		bson.M{"id": sessionID},
		bson.M{"$set": updates},
	)
	return err
}

// DeleteExpiredSessions 删除过期的上传会话
func (r *ResourceRepo) DeleteExpiredSessions(ctx context.Context) error {
	var session resource.UploadSession
	_, err := r.collection.Database().Collection(session.Collection()).DeleteMany(
		ctx,
		bson.M{"expires_at": bson.M{"$lt": time.Now()}},
	)
	return err
}
