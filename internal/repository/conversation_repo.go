package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model"
)

// ConversationRepo 对话仓库
type ConversationRepo struct {
	collection *mongo.Collection
}

// NewConversationRepo 创建对话仓库
func NewConversationRepo(db *mongo.Database) *ConversationRepo {
	return &ConversationRepo{
		collection: db.Collection("conversations"),
	}
}

// Create 创建对话
func (r *ConversationRepo) Create(ctx context.Context, conv *model.Conversation) error {
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, conv)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		conv.ID = oid
	}
	return nil
}

// FindByID 根据 ID 查询
func (r *ConversationRepo) FindByID(ctx context.Context, id string) (*model.Conversation, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var conv model.Conversation
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&conv)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

// AppendMessage 追加消息
func (r *ConversationRepo) AppendMessage(ctx context.Context, id string, msg model.Message) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$push": bson.M{"messages": msg},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err = r.collection.UpdateByID(ctx, objectID, update)
	return err
}

// ListByUserID 查询用户对话列表
func (r *ConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int64) ([]*model.Conversation, error) {
	opts := options.Find().
		SetSort(bson.D{bson.E{Key: "updated_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset).
		SetProjection(bson.M{"messages": 0})

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var convs []*model.Conversation
	if err := cursor.All(ctx, &convs); err != nil {
		return nil, err
	}

	return convs, nil
}

// Delete 删除对话
func (r *ConversationRepo) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// Update 更新对话
func (r *ConversationRepo) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	if setDoc, ok := update["$set"].(bson.M); ok {
		setDoc["updated_at"] = time.Now()
	} else {
		update["$set"] = bson.M{"updated_at": time.Now()}
	}
	_, err = r.collection.UpdateByID(ctx, objectID, update)
	return err
}
