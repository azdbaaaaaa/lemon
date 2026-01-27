# ID类型设计指南

## 问题分析

当前使用 `primitive.ObjectID` 作为ID类型，存在以下问题：
1. **转换麻烦**：在Repository层需要频繁进行 `ObjectIDFromHex()` 和 `.Hex()` 转换
2. **业务层耦合**：业务层（Service/Handler）需要了解MongoDB的ObjectID类型
3. **代码重复**：每个Repository方法都需要重复转换逻辑

## 方案对比

### 方案1: Model层使用string，Repository层封装转换（推荐⭐）

**优点**：
- ✅ 业务层完全使用string，无需转换
- ✅ 保持MongoDB ObjectID的性能优势
- ✅ 转换逻辑集中在Repository层
- ✅ 前端友好（直接使用string）

**缺点**：
- ⚠️ Repository层需要处理转换（但可以封装）

**实现方式**：
```go
// Model层：使用string
type User struct {
    ID       string    `bson:"_id,omitempty" json:"id"`
    UserID   string    `bson:"user_id" json:"user_id"`  // 关联字段也用string
    // ...
}

// Repository层：封装转换逻辑
type UserRepo struct {
    collection *mongo.Collection
}

// 辅助方法：string -> ObjectID
func (r *UserRepo) toObjectID(id string) (primitive.ObjectID, error) {
    return primitive.ObjectIDFromHex(id)
}

// 辅助方法：ObjectID -> string
func (r *UserRepo) toStringID(oid primitive.ObjectID) string {
    return oid.Hex()
}

// 使用示例
func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
    objectID, err := r.toObjectID(id)
    if err != nil {
        return nil, err
    }
    
    var user model.User
    err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
    if err != nil {
        return nil, err
    }
    
    // 自动转换：ObjectID -> string（通过BSON tag）
    return &user, nil
}
```

### 方案2: 使用UUID，完全避免ObjectID

**优点**：
- ✅ 完全不需要转换
- ✅ 标准格式，通用性好
- ✅ 业务层和Repository层都使用string
- ✅ 跨数据库兼容性好

**缺点**：
- ⚠️ UUID较长（36字符）
- ⚠️ 需要配置MongoDB使用string作为_id
- ⚠️ 失去ObjectID的时间戳信息

**实现方式**：
```go
// Model层：使用string存储UUID
type User struct {
    ID       string    `bson:"_id,omitempty" json:"id"`  // UUID格式
    UserID   string    `bson:"user_id" json:"user_id"`   // UUID格式
    // ...
}

// Service层：生成UUID
import "github.com/google/uuid"

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    user := &model.User{
        ID:       uuid.New().String(),  // 直接生成UUID
        Username: req.Username,
        // ...
    }
    
    return s.repo.Create(ctx, user)
}

// Repository层：直接使用string，无需转换
func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
    var user model.User
    err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

### 方案3: 创建ID工具包，简化转换

**优点**：
- ✅ 减少重复代码
- ✅ 保持ObjectID性能优势
- ✅ 统一转换逻辑

**缺点**：
- ⚠️ 仍然需要转换（但更简洁）

**实现方式**：
```go
// internal/pkg/id/id.go
package id

import "go.mongodb.org/mongo-driver/bson/primitive"

// ToObjectID 将string转换为ObjectID
func ToObjectID(id string) (primitive.ObjectID, error) {
    return primitive.ObjectIDFromHex(id)
}

// ToString 将ObjectID转换为string
func ToString(oid primitive.ObjectID) string {
    return oid.Hex()
}

// New 生成新的ObjectID并返回string
func New() string {
    return primitive.NewObjectID().Hex()
}
```

## 推荐方案：方案1（string + Repository封装）

### 理由

1. **最佳实践**：业务层使用string，数据层处理转换
2. **性能优势**：保持MongoDB ObjectID的性能
3. **前端友好**：直接使用string，无需特殊处理
4. **代码清晰**：转换逻辑集中在Repository层

### 完整实现示例

#### 1. Model层定义（使用string）

```go
// internal/model/auth/user.go
package auth

type User struct {
    ID          string     `bson:"_id,omitempty" json:"id"`           // string类型
    Username    string     `bson:"username" json:"username"`
    Email       string     `bson:"email" json:"email"`
    Password    string     `bson:"password" json:"-"`
    Role        UserRole   `bson:"role" json:"role"`
    Status      UserStatus `bson:"status" json:"status"`
    Profile     *UserProfile `bson:"profile,omitempty" json:"profile,omitempty"`
    LastLoginAt *time.Time   `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
    CreatedAt   time.Time    `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time    `bson:"updated_at" json:"updated_at"`
}
```

#### 2. Repository层封装转换逻辑

```go
// internal/repository/user_repo.go
package repository

import (
    "context"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    
    "lemon/internal/model/auth"
)

type UserRepo struct {
    collection *mongo.Collection
}

func NewUserRepo(db *mongo.Database) *UserRepo {
    return &UserRepo{
        collection: db.Collection("users"),
    }
}

// 辅助方法：string -> ObjectID
func (r *UserRepo) toObjectID(id string) (primitive.ObjectID, error) {
    if id == "" {
        return primitive.NilObjectID, nil
    }
    return primitive.ObjectIDFromHex(id)
}

// 辅助方法：ObjectID -> string（通常不需要，BSON会自动转换）
// 但如果需要手动转换，可以使用：
func (r *UserRepo) toStringID(oid primitive.ObjectID) string {
    if oid.IsZero() {
        return ""
    }
    return oid.Hex()
}

// Create 创建用户
func (r *UserRepo) Create(ctx context.Context, user *auth.User) error {
    user.CreatedAt = time.Now()
    user.UpdatedAt = time.Now()
    
    // 如果没有ID，生成新的ObjectID
    if user.ID == "" {
        user.ID = primitive.NewObjectID().Hex()
    }
    
    // 转换为ObjectID存储
    objectID, err := r.toObjectID(user.ID)
    if err != nil {
        return err
    }
    
    // 创建临时结构体用于存储（_id使用ObjectID）
    doc := bson.M{
        "_id":        objectID,
        "username":   user.Username,
        "email":      user.Email,
        "password":   user.Password,
        "role":       user.Role,
        "status":     user.Status,
        "profile":    user.Profile,
        "created_at": user.CreatedAt,
        "updated_at": user.UpdatedAt,
    }
    if user.LastLoginAt != nil {
        doc["last_login_at"] = *user.LastLoginAt
    }
    
    _, err = r.collection.InsertOne(ctx, doc)
    if err != nil {
        return err
    }
    
    return nil
}

// FindByID 根据ID查询
func (r *UserRepo) FindByID(ctx context.Context, id string) (*auth.User, error) {
    objectID, err := r.toObjectID(id)
    if err != nil {
        return nil, err
    }
    
    // 使用BSON解码，自动处理ObjectID -> string转换
    var doc bson.M
    err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
    if err != nil {
        return nil, err
    }
    
    // 手动转换（因为_id是ObjectID，需要转换为string）
    user := &auth.User{
        ID:        doc["_id"].(primitive.ObjectID).Hex(),
        Username:  doc["username"].(string),
        Email:     doc["email"].(string),
        Role:      auth.UserRole(doc["role"].(string)),
        Status:    auth.UserStatus(doc["status"].(string)),
        CreatedAt: doc["created_at"].(time.Time),
        UpdatedAt: doc["updated_at"].(time.Time),
    }
    
    if profile, ok := doc["profile"].(bson.M); ok && profile != nil {
        user.Profile = &auth.UserProfile{
            Nickname: getString(profile, "nickname"),
            Avatar:   getString(profile, "avatar"),
            Phone:    getString(profile, "phone"),
        }
    }
    
    if lastLoginAt, ok := doc["last_login_at"].(time.Time); ok {
        user.LastLoginAt = &lastLoginAt
    }
    
    return user, nil
}

// 更优雅的方式：使用自定义BSON编解码器
// 见下面的"高级优化"部分
```

#### 3. 高级优化：自定义BSON编解码器（可选）

如果觉得手动转换太麻烦，可以实现自定义的BSON编解码器：

```go
// internal/pkg/mongodb/id_codec.go
package mongodb

import (
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/bson/bsoncodec"
    "go.mongodb.org/mongo-driver/bson/bsonrw"
)

// StringID 自定义ID类型，自动处理ObjectID转换
type StringID string

// MarshalBSONValue 编码：string -> ObjectID
func (id StringID) MarshalBSONValue() (bsontype.Type, []byte, error) {
    if id == "" {
        oid := primitive.NewObjectID()
        return bson.TypeObjectID, oid[:], nil
    }
    oid, err := primitive.ObjectIDFromHex(string(id))
    if err != nil {
        return 0, nil, err
    }
    return bson.TypeObjectID, oid[:], nil
}

// UnmarshalBSONValue 解码：ObjectID -> string
func (id *StringID) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
    if t != bson.TypeObjectID {
        return errors.New("invalid type for StringID")
    }
    var oid primitive.ObjectID
    copy(oid[:], data)
    *id = StringID(oid.Hex())
    return nil
}

// 使用方式
type User struct {
    ID StringID `bson:"_id,omitempty" json:"id"`  // 自动处理转换
    // ...
}
```

## 最终推荐

**推荐使用方案1（string + Repository封装）**，原因：

1. ✅ **简单直接**：业务层完全使用string，无需关心底层实现
2. ✅ **性能优秀**：保持MongoDB ObjectID的性能优势
3. ✅ **前端友好**：直接使用string，无需特殊处理
4. ✅ **代码清晰**：转换逻辑集中在Repository层，易于维护

**如果觉得Repository层转换仍然麻烦，可以考虑：**
- 使用自定义BSON编解码器（高级优化）
- 或者直接使用UUID（方案2），完全避免转换

## 迁移建议

如果要从ObjectID迁移到string：

1. **第一步**：修改Model定义，将ID字段改为string
2. **第二步**：修改Repository层，添加转换辅助方法
3. **第三步**：更新所有使用ID的地方（Service/Handler）
4. **第四步**：测试验证

## 注意事项

1. **索引**：MongoDB的_id字段会自动创建索引，使用string作为_id时性能仍然很好
2. **查询性能**：ObjectID查询性能略优于string，但差异很小
3. **数据迁移**：如果已有数据使用ObjectID，需要迁移脚本
4. **关联查询**：关联字段（如user_id）也需要统一使用string
