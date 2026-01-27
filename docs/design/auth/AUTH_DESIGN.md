# 认证与权限系统设计文档

## 1. 系统概述

### 1.1 功能需求

- **用户认证**: 基于JWT的认证机制，支持Access Token和Refresh Token
- **用户角色**: 三种角色（超级管理员、编辑人员、审核人员）
- **权限控制**: 基于角色的访问控制（RBAC）
- **用户管理**: 支持用户注册和管理员创建用户

### 1.2 角色定义

| 角色 | 角色代码 | 权限说明 |
|------|----------|----------|
| 超级管理员 | `admin` | 可以查看所有数据和操作所有数据的权限 |
| 编辑人员 | `editor` | 可以上传小说等，创建工作流，管理自己的内容 |
| 审核人员 | `reviewer` | 可以对内容进行审核，查看待审核内容 |

## 2. 数据模型设计

### 2.1 User实体

```go
// internal/model/user.go
type User struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Username    string             `bson:"username" json:"username"`           // 用户名（唯一）
    Email       string             `bson:"email" json:"email"`                 // 邮箱（唯一）
    Password    string             `bson:"password" json:"-"`                  // 密码（加密存储，不返回）
    Role        UserRole           `bson:"role" json:"role"`                   // 角色
    Status      UserStatus         `bson:"status" json:"status"`               // 状态
    Profile     *UserProfile       `bson:"profile,omitempty" json:"profile,omitempty"` // 用户资料
    LastLoginAt *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type UserProfile struct {
    Nickname string `bson:"nickname,omitempty" json:"nickname,omitempty"`
    Avatar   string `bson:"avatar,omitempty" json:"avatar,omitempty"`
    Phone    string `bson:"phone,omitempty" json:"phone,omitempty"`
}

type UserRole string

const (
    RoleAdmin    UserRole = "admin"    // 超级管理员
    RoleEditor   UserRole = "editor"   // 编辑人员
    RoleReviewer UserRole = "reviewer" // 审核人员
)

type UserStatus string

const (
    UserStatusActive   UserStatus = "active"   // 激活
    UserStatusInactive UserStatus = "inactive" // 未激活（注册待审核）
    UserStatusBanned   UserStatus = "banned"   // 禁用
)
```

### 2.2 RefreshToken实体

```go
type RefreshToken struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
    Token     string             `bson:"token" json:"token"` // Refresh Token值
    ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
    CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
```

## 3. API接口设计

### 3.1 认证相关接口

#### 3.1.1 用户注册

**POST** `/api/v1/auth/register`

**请求体**:
```json
{
  "username": "editor001",
  "email": "editor@example.com",
  "password": "123456",
  "nickname": "编辑小王"
}
```

**响应** (201 Created):
```json
{
  "code": 0,
  "message": "注册成功，等待管理员审核",
  "data": {
    "user_id": "507f1f77bcf86cd799439011",
    "username": "editor001",
    "status": "inactive"
  }
}
```

#### 3.1.2 用户登录

**POST** `/api/v1/auth/login`

**请求体**:
```json
{
  "username": "editor001",
  "password": "123456"
}
```

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "token_type": "Bearer",
    "user": {
      "id": "507f1f77bcf86cd799439011",
      "username": "editor001",
      "email": "editor@example.com",
      "role": "editor",
      "status": "active"
    }
  }
}
```

#### 3.1.3 刷新Token

**POST** `/api/v1/auth/refresh`

**请求体**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "刷新成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "token_type": "Bearer"
  }
}
```

#### 3.1.4 退出登录

**POST** `/api/v1/auth/logout`

**请求头**: `Authorization: Bearer {access_token}`

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "退出成功"
}
```

#### 3.1.5 获取当前用户信息

**GET** `/api/v1/auth/me`

**请求头**: `Authorization: Bearer {access_token}`

**响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "username": "editor001",
    "email": "editor@example.com",
    "role": "editor",
    "status": "active",
    "profile": {
      "nickname": "编辑小王",
      "avatar": "https://..."
    },
    "last_login_at": "2024-01-01T12:00:00Z"
  }
}
```

### 3.2 用户管理接口（管理员）

#### 3.2.1 创建用户

**POST** `/api/v1/users`

**权限**: 仅管理员

**请求体**:
```json
{
  "username": "reviewer001",
  "email": "reviewer@example.com",
  "password": "123456",
  "role": "reviewer",
  "status": "active",
  "profile": {
    "nickname": "审核员小李"
  }
}
```

**响应** (201 Created):
```json
{
  "code": 0,
  "message": "用户创建成功",
  "data": {
    "id": "507f1f77bcf86cd799439012",
    "username": "reviewer001",
    "email": "reviewer@example.com",
    "role": "reviewer",
    "status": "active"
  }
}
```

#### 3.2.2 查询用户列表

**GET** `/api/v1/users`

**权限**: 仅管理员

**查询参数**:
- `page`: 页码 (默认: 1)
- `page_size`: 每页数量 (默认: 20)
- `role`: 角色筛选
- `status`: 状态筛选
- `keyword`: 关键词搜索（用户名、邮箱）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "users": [
      {
        "id": "507f1f77bcf86cd799439011",
        "username": "editor001",
        "email": "editor@example.com",
        "role": "editor",
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

#### 3.2.3 查询用户详情

**GET** `/api/v1/users/:id`

**权限**: 管理员或本人

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "username": "editor001",
    "email": "editor@example.com",
    "role": "editor",
    "status": "active",
    "profile": {
      "nickname": "编辑小王"
    },
    "created_at": "2024-01-01T00:00:00Z",
    "last_login_at": "2024-01-01T12:00:00Z"
  }
}
```

#### 3.2.4 更新用户

**PUT** `/api/v1/users/:id`

**权限**: 管理员或本人（本人只能更新部分字段）

**请求体**:
```json
{
  "profile": {
    "nickname": "新昵称",
    "avatar": "https://..."
  }
}
```

**管理员可更新字段**:
```json
{
  "role": "editor",
  "status": "active",
  "profile": {...}
}
```

#### 3.2.5 审核用户（激活/禁用）

**POST** `/api/v1/users/:id/approve`

**权限**: 仅管理员

**请求体**:
```json
{
  "action": "approve",  // approve: 激活, reject: 拒绝
  "reason": "审核通过"
}
```

#### 3.2.6 删除用户

**DELETE** `/api/v1/users/:id`

**权限**: 仅管理员

**响应**:
```json
{
  "code": 0,
  "message": "用户删除成功"
}
```

#### 3.2.7 修改密码

**POST** `/api/v1/users/:id/password`

**权限**: 管理员或本人

**请求体**:
```json
{
  "old_password": "旧密码",
  "new_password": "新密码"
}
```

## 4. JWT Token设计

### 4.1 Access Token

**Payload结构**:
```json
{
  "user_id": "507f1f77bcf86cd799439011",
  "username": "editor001",
  "role": "editor",
  "exp": 1704067200,
  "iat": 1704063600
}
```

**配置**:
- 过期时间: 1小时
- 签名算法: HS256
- Secret: 从配置文件读取

### 4.2 Refresh Token

**存储方式**: MongoDB

**配置**:
- 过期时间: 7天
- 长度: 32字节随机字符串
- 存储: 数据库 + Redis缓存

## 5. 权限控制设计

### 5.1 权限定义

```go
// internal/pkg/auth/permissions.go

const (
    // 工作流权限
    PermissionWorkflowCreate   = "workflow:create"
    PermissionWorkflowView      = "workflow:view"
    PermissionWorkflowViewAll   = "workflow:view:all"  // 查看所有工作流
    PermissionWorkflowEdit     = "workflow:edit"
    PermissionWorkflowDelete   = "workflow:delete"
    PermissionWorkflowManage   = "workflow:manage"    // 管理所有工作流
    
    // 用户管理权限
    PermissionUserCreate       = "user:create"
    PermissionUserView         = "user:view"
    PermissionUserViewAll      = "user:view:all"
    PermissionUserEdit         = "user:edit"
    PermissionUserDelete       = "user:delete"
    PermissionUserApprove      = "user:approve"
    
    // 审核权限
    PermissionReview            = "review:review"
    PermissionReviewView       = "review:view"
)

// 角色权限映射
var RolePermissions = map[UserRole][]string{
    RoleAdmin: {
        PermissionWorkflowCreate,
        PermissionWorkflowView,
        PermissionWorkflowViewAll,
        PermissionWorkflowEdit,
        PermissionWorkflowDelete,
        PermissionWorkflowManage,
        PermissionUserCreate,
        PermissionUserView,
        PermissionUserViewAll,
        PermissionUserEdit,
        PermissionUserDelete,
        PermissionUserApprove,
        PermissionReview,
        PermissionReviewView,
    },
    RoleEditor: {
        PermissionWorkflowCreate,
        PermissionWorkflowView,  // 只能查看自己的
        PermissionWorkflowEdit,  // 只能编辑自己的
        PermissionWorkflowDelete, // 只能删除自己的
    },
    RoleReviewer: {
        PermissionWorkflowView,
        PermissionWorkflowViewAll,
        PermissionReview,
        PermissionReviewView,
    },
}
```

### 5.2 中间件设计

#### 5.2.1 JWT认证中间件

```go
// internal/server/middleware/auth.go
func Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从Header获取Token
        token := extractToken(c)
        if token == "" {
            c.JSON(401, gin.H{"code": 40101, "message": "未授权"})
            c.Abort()
            return
        }
        
        // 验证Token
        claims, err := jwt.ValidateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"code": 40102, "message": "Token无效或已过期"})
            c.Abort()
            return
        }
        
        // 将用户信息存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("role", claims.Role)
        
        c.Next()
    }
}
```

#### 5.2.2 权限检查中间件

```go
// internal/server/middleware/permission.go
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("role")
        if role == "" {
            c.JSON(401, gin.H{"code": 40101, "message": "未授权"})
            c.Abort()
            return
        }
        
        // 检查权限
        if !hasPermission(role, permission) {
            c.JSON(403, gin.H{"code": 40301, "message": "无权限访问"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// 检查角色
func RequireRole(roles ...UserRole) gin.HandlerFunc {
    return func(c *gin.Context) {
        roleStr := c.GetString("role")
        if roleStr == "" {
            c.JSON(401, gin.H{"code": 40101, "message": "未授权"})
            c.Abort()
            return
        }
        
        role := UserRole(roleStr)
        for _, r := range roles {
            if role == r {
                c.Next()
                return
            }
        }
        
        c.JSON(403, gin.H{"code": 40301, "message": "无权限访问"})
        c.Abort()
    }
}
```

### 5.3 资源所有权检查

```go
// 检查工作流所有权
func CheckWorkflowOwnership(c *gin.Context, workflowID string) bool {
    userID := c.GetString("user_id")
    role := c.GetString("role")
    
    // 管理员可以访问所有
    if role == string(RoleAdmin) {
        return true
    }
    
    // 查询工作流所有者
    workflow, err := workflowRepo.GetByID(workflowID)
    if err != nil {
        return false
    }
    
    return workflow.UserID == userID
}
```

## 6. 路由保护示例

```go
// 公开路由
public := v1.Group("")
{
    public.POST("/auth/register", authHandler.Register)
    public.POST("/auth/login", authHandler.Login)
    public.POST("/auth/refresh", authHandler.Refresh)
}

// 需要认证的路由
auth := v1.Group("")
auth.Use(middleware.Auth())
{
    auth.POST("/auth/logout", authHandler.Logout)
    auth.GET("/auth/me", authHandler.GetMe)
    
    // 工作流路由
    workflow := auth.Group("/workflow")
    {
        workflow.POST("", workflowHandler.Create) // 需要 workflow:create 权限
        workflow.GET("", workflowHandler.List)     // 需要 workflow:view 权限
        workflow.GET("/:id", workflowHandler.Get)  // 需要 workflow:view 权限 + 所有权检查
    }
}

// 仅管理员路由
admin := v1.Group("")
admin.Use(middleware.Auth())
admin.Use(middleware.RequireRole(RoleAdmin))
{
    admin.POST("/users", userHandler.Create)
    admin.GET("/users", userHandler.List)
    admin.PUT("/users/:id", userHandler.Update)
    admin.DELETE("/users/:id", userHandler.Delete)
    admin.POST("/users/:id/approve", userHandler.Approve)
}
```

## 7. 密码加密

使用bcrypt加密密码：

```go
import "golang.org/x/crypto/bcrypt"

// 加密密码
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

// 验证密码
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

## 8. 错误码定义

| 错误码 | 说明 |
|--------|------|
| 40101 | 未授权（未提供Token） |
| 40102 | Token无效或已过期 |
| 40103 | Refresh Token无效或已过期 |
| 40301 | 无权限访问 |
| 40302 | 无权限操作该资源 |
| 40001 | 用户名已存在 |
| 40002 | 邮箱已存在 |
| 40003 | 密码格式不正确 |
| 40004 | 用户名或密码错误 |
| 40005 | 用户未激活 |
| 40006 | 用户已被禁用 |

## 9. 安全考虑

### 9.1 Token安全

- Access Token存储在内存中（前端），不存储在localStorage
- Refresh Token存储在HttpOnly Cookie中（推荐）或安全存储
- Token过期后自动刷新
- 支持Token撤销（黑名单机制）

### 9.2 密码安全

- 密码最小长度6位
- 密码使用bcrypt加密存储
- 登录失败次数限制（可选）
- 密码修改需要验证旧密码

### 9.3 其他安全措施

- CORS配置
- 请求频率限制
- SQL注入防护（使用参数化查询）
- XSS防护

## 10. 数据库索引

```go
// users集合索引
{
    "username": 1  // 唯一索引
}
{
    "email": 1     // 唯一索引
}
{
    "role": 1,
    "status": 1
}

// refresh_tokens集合索引
{
    "user_id": 1
}
{
    "token": 1     // 唯一索引
}
{
    "expires_at": 1  // TTL索引，自动删除过期token
}
```
