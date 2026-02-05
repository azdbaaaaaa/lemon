package novel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// ShotRepository 镜头仓库接口
type ShotRepository interface {
	Create(ctx context.Context, shot *novel.Shot) error
	CreateMany(ctx context.Context, shots []*novel.Shot) error
	FindByID(ctx context.Context, id string) (*novel.Shot, error)
	FindBySceneID(ctx context.Context, sceneID string) ([]*novel.Shot, error)
	FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Shot, error)
	FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Shot, error)
	FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Shot, error)
	FindByChapterIDAndSceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.Shot, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error
	Delete(ctx context.Context, id string) error
	DeleteBySceneID(ctx context.Context, sceneID string) error
	DeleteByNarrationID(ctx context.Context, narrationID string) error
}

// ShotRepo 镜头仓库实现
type ShotRepo struct {
	coll *mongo.Collection
}

// NewShotRepo 创建镜头仓库
func NewShotRepo(db *mongo.Database) *ShotRepo {
	var s novel.Shot
	return &ShotRepo{coll: db.Collection(s.Collection())}
}

// Create 创建镜头
func (r *ShotRepo) Create(ctx context.Context, shot *novel.Shot) error {
	now := time.Now()
	shot.CreatedAt = now
	shot.UpdatedAt = now
	if shot.Status == "" || shot.Status == novel.TaskStatus("") {
		shot.Status = novel.TaskStatusCompleted
	}
	if shot.Version == 0 {
		shot.Version = 1
	}
	_, err := r.coll.InsertOne(ctx, shot)
	return err
}

// CreateMany 批量创建镜头
func (r *ShotRepo) CreateMany(ctx context.Context, shots []*novel.Shot) error {
	if len(shots) == 0 {
		return nil
	}
	now := time.Now()
	docs := make([]interface{}, len(shots))
	for i, shot := range shots {
		shot.CreatedAt = now
		shot.UpdatedAt = now
		if shot.Status == "" {
			shot.Status = "completed"
		}
		if shot.Version == 0 {
			shot.Version = 1
		}
		docs[i] = shot
	}
	_, err := r.coll.InsertMany(ctx, docs)
	return err
}

// FindByID 根据ID查询镜头
func (r *ShotRepo) FindByID(ctx context.Context, id string) (*novel.Shot, error) {
	var shot novel.Shot
	if err := r.coll.FindOne(ctx, bson.M{"id": id, "deleted_at": nil}).Decode(&shot); err != nil {
		return nil, err
	}
	return &shot, nil
}

// FindBySceneID 根据场景ID查询所有镜头（按sequence排序）
func (r *ShotRepo) FindBySceneID(ctx context.Context, sceneID string) ([]*novel.Shot, error) {
	filter := bson.M{"scene_id": sceneID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"sequence": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var shots []*novel.Shot
	if err := cur.All(ctx, &shots); err != nil {
		return nil, err
	}
	return shots, nil
}

// FindByNarrationID 根据解说ID查询所有镜头（按index排序）
func (r *ShotRepo) FindByNarrationID(ctx context.Context, narrationID string) ([]*novel.Shot, error) {
	filter := bson.M{"narration_id": narrationID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"index": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var shots []*novel.Shot
	if err := cur.All(ctx, &shots); err != nil {
		return nil, err
	}
	return shots, nil
}

// FindByNarrationIDAndVersion 根据解说ID和版本号查询镜头
func (r *ShotRepo) FindByNarrationIDAndVersion(ctx context.Context, narrationID string, version int) ([]*novel.Shot, error) {
	filter := bson.M{"narration_id": narrationID, "version": version, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"index": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var shots []*novel.Shot
	if err := cur.All(ctx, &shots); err != nil {
		return nil, err
	}
	return shots, nil
}

// FindByChapterID 根据章节ID查询所有镜头
func (r *ShotRepo) FindByChapterID(ctx context.Context, chapterID string) ([]*novel.Shot, error) {
	filter := bson.M{"chapter_id": chapterID, "deleted_at": nil}
	opts := options.Find().SetSort(bson.M{"index": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var shots []*novel.Shot
	if err := cur.All(ctx, &shots); err != nil {
		return nil, err
	}
	return shots, nil
}

// FindByChapterIDAndSceneAndShot 根据章节ID、场景编号和镜头编号查询镜头
func (r *ShotRepo) FindByChapterIDAndSceneAndShot(ctx context.Context, chapterID, sceneNumber, shotNumber string) (*novel.Shot, error) {
	var shot novel.Shot
	filter := bson.M{
		"chapter_id":  chapterID,
		"scene_number": sceneNumber,
		"shot_number":  shotNumber,
		"deleted_at":   nil,
	}
	if err := r.coll.FindOne(ctx, filter).Decode(&shot); err != nil {
		return nil, err
	}
	return &shot, nil
}

// Update 更新镜头
func (r *ShotRepo) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": updates},
	)
	return err
}

// UpdateStatus 更新镜头状态
func (r *ShotRepo) UpdateStatus(ctx context.Context, id string, status novel.TaskStatus, errorMessage string) error {
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

// Delete 软删除镜头
func (r *ShotRepo) Delete(ctx context.Context, id string) error {
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

// DeleteBySceneID 根据场景ID软删除所有镜头
func (r *ShotRepo) DeleteBySceneID(ctx context.Context, sceneID string) error {
	_, err := r.coll.UpdateMany(
		ctx,
		bson.M{"scene_id": sceneID, "deleted_at": nil},
		bson.M{"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)
	return err
}

// DeleteByNarrationID 根据解说ID软删除所有镜头
func (r *ShotRepo) DeleteByNarrationID(ctx context.Context, narrationID string) error {
	_, err := r.coll.UpdateMany(
		ctx,
		bson.M{"narration_id": narrationID, "deleted_at": nil},
		bson.M{"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)
	return err
}
