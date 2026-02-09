package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ImageRepository 图片仓库接口（供 service 层依赖）
type ImageRepository interface {
	Create(ctx context.Context, image *novel.Image) error
	FindByID(ctx context.Context, id string) (*novel.Image, error)
	FindByCharacterID(ctx context.Context, characterID string) ([]*novel.Image, error)                                                // 查询角色的所有图片
	FindByCharacterIDAndSubtype(ctx context.Context, characterID string, subtype novel.CharacterImageSubtype) ([]*novel.Image, error) // 查询角色的指定细分类图片
	FindByPropID(ctx context.Context, propID string) ([]*novel.Image, error)                                                          // 查询道具的所有图片
	FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, imageType novel.ImageType, version int) ([]*novel.Image, error) // 查询镜头的首图/尾图
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error
	Delete(ctx context.Context, id string) error
}

// ImageRepo 图片仓库
type ImageRepo struct {
	coll *mongo.Collection
}

// NewImageRepo 创建图片仓库
func NewImageRepo(db *mongo.Database) *ImageRepo {
	var i novel.Image
	return &ImageRepo{coll: db.Collection(i.Collection())}
}

// Create 创建图片记录
func (r *ImageRepo) Create(ctx context.Context, image *novel.Image) error {
	now := time.Now()
	image.CreatedAt = now
	image.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, image)
	return err
}

// FindByID 根据ID查询
func (r *ImageRepo) FindByID(ctx context.Context, id string) (*novel.Image, error) {
	var image novel.Image
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

// FindByCharacterID 查询角色的所有图片
func (r *ImageRepo) FindByCharacterID(ctx context.Context, characterID string) ([]*novel.Image, error) {
	filter := bson.M{"character_id": characterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByCharacterIDAndSubtype 查询角色的指定细分类图片
func (r *ImageRepo) FindByCharacterIDAndSubtype(ctx context.Context, characterID string, subtype novel.CharacterImageSubtype) ([]*novel.Image, error) {
	filter := bson.M{
		"character_id":            characterID,
		"character_image_subtype": subtype,
		"deleted_at":              nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByPropID 查询道具的所有图片
func (r *ImageRepo) FindByPropID(ctx context.Context, propID string) ([]*novel.Image, error) {
	filter := bson.M{"prop_id": propID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// FindByShotIDAndTypeAndVersion 根据镜头ID、图片类型和版本号查询图片
func (r *ImageRepo) FindByShotIDAndTypeAndVersion(ctx context.Context, shotID string, imageType novel.ImageType, version int) ([]*novel.Image, error) {
	filter := bson.M{
		"shot_id":    shotID,
		"image_type": imageType,
		"version":    version,
		"deleted_at": nil,
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []*novel.Image
	if err := cur.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
}

// UpdateStatus 更新状态
func (r *ImageRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus) error {
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

// Delete 软删除
func (r *ImageRepo) Delete(ctx context.Context, id string) error {
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
