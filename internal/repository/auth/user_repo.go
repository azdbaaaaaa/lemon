package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/auth"
)

// UserRepo 用户仓库
// 使用UUID作为ID，无需ObjectID转换
type UserRepo struct {
	collection *mongo.Collection
}

// NewUserRepo 创建用户仓库
func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{
		collection: db.Collection("users"),
	}
}

// Create 创建用户
func (r *UserRepo) Create(ctx context.Context, user *auth.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// 直接使用string ID，无需转换
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

// FindByID 根据ID查询用户
func (r *UserRepo) FindByID(ctx context.Context, id string) (*auth.User, error) {
	// 直接使用string ID查询，无需转换
	var user auth.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername 根据用户名查询用户
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	var user auth.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 根据邮箱查询用户
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	var user auth.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *UserRepo) Update(ctx context.Context, id string, update bson.M) error {
	// 自动更新updated_at
	if setDoc, ok := update["$set"].(bson.M); ok {
		setDoc["updated_at"] = time.Now()
	} else {
		update["$set"] = bson.M{"updated_at": time.Now()}
	}

	// 直接使用string ID，无需转换
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// UpdateLastLoginAt 更新最后登录时间
func (r *UserRepo) UpdateLastLoginAt(ctx context.Context, id string) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"last_login_at": now,
			"updated_at":    now,
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// Delete 删除用户
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	// 直接使用string ID，无需转换
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// List 查询用户列表（支持分页和筛选）
func (r *UserRepo) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*auth.User, int64, error) {
	// 计算总数
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 分页选项
	opts := options.Find().
		SetSort(bson.D{bson.E{Key: "created_at", Value: -1}}).
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*auth.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Count 统计用户数量
func (r *UserRepo) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}
