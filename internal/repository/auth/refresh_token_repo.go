package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"lemon/internal/model/auth"
)

// RefreshTokenRepo RefreshToken仓库
// 使用UUID作为ID，无需ObjectID转换
type RefreshTokenRepo struct {
	collection *mongo.Collection
}

// NewRefreshTokenRepo 创建RefreshToken仓库
func NewRefreshTokenRepo(db *mongo.Database) *RefreshTokenRepo {
	return &RefreshTokenRepo{
		collection: db.Collection("refresh_tokens"),
	}
}

// Create 创建RefreshToken
func (r *RefreshTokenRepo) Create(ctx context.Context, token *auth.RefreshToken) error {
	token.CreatedAt = time.Now()

	// 直接使用string ID，无需转换
	_, err := r.collection.InsertOne(ctx, token)
	return err
}

// FindByToken 根据Token值查询
func (r *RefreshTokenRepo) FindByToken(ctx context.Context, token string) (*auth.RefreshToken, error) {
	var refreshToken auth.RefreshToken
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&refreshToken)
	if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}

// FindByUserID 根据UserID查询所有Token
func (r *RefreshTokenRepo) FindByUserID(ctx context.Context, userID string) ([]*auth.RefreshToken, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tokens []*auth.RefreshToken
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// Delete 删除指定的Token
func (r *RefreshTokenRepo) Delete(ctx context.Context, id string) error {
	// 直接使用string ID，无需转换
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteByToken 根据Token值删除
func (r *RefreshTokenRepo) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"token": token})
	return err
}

// DeleteByUserID 删除用户的所有Token
func (r *RefreshTokenRepo) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}

// DeleteExpired 删除所有过期的Token
func (r *RefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	_, err := r.collection.DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lt": now}})
	return err
}

// Exists 检查Token是否存在
func (r *RefreshTokenRepo) Exists(ctx context.Context, token string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"token": token})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
