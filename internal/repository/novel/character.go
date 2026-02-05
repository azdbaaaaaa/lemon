package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// CharacterRepository 角色仓库接口（供 service 层依赖）
type CharacterRepository interface {
	Create(ctx context.Context, character *novel.Character) error
	FindByID(ctx context.Context, id string) (*novel.Character, error)
	FindByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error)
	FindByNameAndNovelID(ctx context.Context, name, novelID string) (*novel.Character, error)
	Update(ctx context.Context, id string, updates bson.M) error
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error
	Delete(ctx context.Context, id string) error
}

// CharacterRepo 角色仓库
type CharacterRepo struct {
	coll *mongo.Collection
}

// NewCharacterRepo 创建角色仓库
func NewCharacterRepo(db *mongo.Database) *CharacterRepo {
	var c novel.Character
	return &CharacterRepo{coll: db.Collection(c.Collection())}
}

// Create 创建角色
func (r *CharacterRepo) Create(ctx context.Context, character *novel.Character) error {
	now := time.Now()
	character.CreatedAt = now
	character.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, character)
	return err
}

// FindByID 根据ID查询角色
func (r *CharacterRepo) FindByID(ctx context.Context, id string) (*novel.Character, error) {
	var character novel.Character
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&character); err != nil {
		return nil, err
	}
	return &character, nil
}

// FindByNovelID 查询小说的所有角色
func (r *CharacterRepo) FindByNovelID(ctx context.Context, novelID string) ([]*novel.Character, error) {
	filter := bson.M{"novel_id": novelID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var characters []*novel.Character
	if err := cur.All(ctx, &characters); err != nil {
		return nil, err
	}
	return characters, nil
}

// FindByNameAndNovelID 根据名称和小说ID查询（用于去重）
func (r *CharacterRepo) FindByNameAndNovelID(ctx context.Context, name, novelID string) (*novel.Character, error) {
	var character novel.Character
	filter := bson.M{"name": name, "novel_id": novelID, "deleted_at": nil}
	if err := r.coll.FindOne(ctx, filter).Decode(&character); err != nil {
		return nil, err
	}
	return &character, nil
}

// Update 更新角色信息
func (r *CharacterRepo) Update(ctx context.Context, id string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": updates},
	)
	return err
}

// UpdateStatus 更新角色状态
func (r *CharacterRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error {
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

// Delete 软删除角色
func (r *CharacterRepo) Delete(ctx context.Context, id string) error {
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
