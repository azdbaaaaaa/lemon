package prompt

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/prompt"
)

// PromptRepository 提示词模板仓库接口
// 封装对 prompt_templates 集合的 CRUD 操作。
type PromptRepository interface {
	Create(ctx context.Context, tpl *prompt.PromptTemplate) error
	Update(ctx context.Context, id string, updates bson.M) error
	FindByID(ctx context.Context, id string) (*prompt.PromptTemplate, error)
	FindByTypeCode(ctx context.Context, promptType, code, language string) (*prompt.PromptTemplate, error)
	ListByType(ctx context.Context, promptType string, page, pageSize int64) ([]*prompt.PromptTemplate, int64, error)
}

// promptRepository 提示词模板仓库实现
type promptRepository struct {
	collection *mongo.Collection
}

// NewPromptRepository 创建提示词模板仓库
func NewPromptRepository(db *mongo.Database) PromptRepository {
	var tpl prompt.PromptTemplate
	return &promptRepository{
		collection: db.Collection(tpl.Collection()),
	}
}

// Create 创建提示词模板
func (r *promptRepository) Create(ctx context.Context, tpl *prompt.PromptTemplate) error {
	now := time.Now()
	tpl.CreatedAt = now
	tpl.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, tpl)
	return err
}

// Update 更新提示词模板
func (r *promptRepository) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"id": id, "deleted_at": nil},
		bson.M{"$set": updates},
	)
	return err
}

// FindByID 根据 ID 查询提示词模板
func (r *promptRepository) FindByID(ctx context.Context, id string) (*prompt.PromptTemplate, error) {
	var tpl prompt.PromptTemplate
	err := r.collection.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&tpl)
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}

// FindByTypeCode 按类型、编码、语言查询提示词模板
// language 为空时，不参与过滤。
func (r *promptRepository) FindByTypeCode(ctx context.Context, promptType, code, language string) (*prompt.PromptTemplate, error) {
	filter := bson.M{
		"type":       promptType,
		"code":       code,
		"deleted_at": nil,
	}
	if language != "" {
		filter["language"] = language
	}

	var tpl prompt.PromptTemplate
	err := r.collection.FindOne(ctx, filter).Decode(&tpl)
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}

// ListByType 按类型分页查询提示词模板
func (r *promptRepository) ListByType(ctx context.Context, promptType string, page, pageSize int64) ([]*prompt.PromptTemplate, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := bson.M{
		"type":       promptType,
		"deleted_at": nil,
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(pageSize).
		SetSkip(skip)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*prompt.PromptTemplate
	if err := cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

