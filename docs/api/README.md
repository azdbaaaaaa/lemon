# Lemon API 接口文档

## 📚 相关文档

- [开发规范](../guide/DEVELOPMENT_RULES.md) - 开发流程和规范
- [认证系统设计](../design/auth/AUTH_DESIGN.md) - 认证模块设计
- [工作流API设计](../design/workflow/API_DESIGN.md) - 工作流API设计

## 快速开始

### 启动服务器

```bash
# 编译项目
go build -o lemon main.go

# 启动服务器（默认端口8080）
./lemon serve

# 或指定端口和模式
./lemon serve --port 8080 --mode debug
```

### 访问 Swagger 接口文档

启动服务器后，在浏览器中访问：

**http://localhost:8080/swagger/index.html**

### 认证流程

1. **注册用户** → `POST /api/v1/auth/register`
2. **管理员审核** → `POST /api/v1/users/approve` (需要管理员权限，id在body中)
3. **用户登录** → `POST /api/v1/auth/login` → 获取 `access_token` 和 `refresh_token`
4. **使用Token** → 在请求头中携带 `Authorization: Bearer {access_token}`
5. **刷新Token** → Token过期后使用 `POST /api/v1/auth/refresh`

## API 接口列表

### 健康检查

- `GET /health` - 健康检查
- `GET /ready` - 就绪检查

### 认证接口

- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新Token
- `POST /api/v1/auth/logout` - 退出登录
- `GET /api/v1/auth/me` - 获取当前用户信息

### 用户管理接口（需要管理员权限）

- `POST /api/v1/users` - 创建用户
- `GET /api/v1/users` - 查询用户列表（支持id参数查询详情）
- `POST /api/v1/users/update` - 更新用户（id在body中）
- `POST /api/v1/users/delete` - 删除用户（id在body中）
- `POST /api/v1/users/approve` - 审核用户（激活/禁用，id在body中）
- `POST /api/v1/users/password` - 修改密码（id在body中）

### 对话接口

- `POST /api/v1/chat` - 对话接口
- `POST /api/v1/chat/stream` - 流式对话接口（SSE）

### 文本转换

- `POST /api/v1/transform` - 文本转换接口（需要配置AI API Key）

### 对话管理

- `POST /api/v1/conversations` - 创建对话
- `GET /api/v1/conversations` - 获取对话列表（user_id参数）或详情（id参数）
- `POST /api/v1/conversations/delete` - 删除对话（id在body中）

## 配置说明

### 环境变量

可以通过环境变量覆盖配置：

- `LEMON_SERVER_PORT` - 服务器端口
- `LEMON_AI_API_KEY` - AI API密钥
- `LEMON_MONGO_URI` - MongoDB连接URI
- `LEMON_REDIS_ADDR` - Redis地址

### 配置文件

默认配置文件：`configs/config.yaml`

## 认证说明

### Token使用

大部分API接口需要认证，需要在请求头中携带Token：

```
Authorization: Bearer {access_token}
```

### Token刷新

Access Token有效期为1小时，过期后可以使用Refresh Token刷新：

```bash
POST /api/v1/auth/refresh
{
  "refresh_token": "{refresh_token}"
}
```

### 角色权限

- **admin**: 超级管理员，拥有所有权限
- **editor**: 编辑人员，可以创建工作流、管理自己的内容
- **reviewer**: 审核人员，可以审核内容、查看所有工作流

## 注意事项

1. MongoDB 和 Redis 是可选的，如果未配置，相关功能将不可用
2. Transform 接口需要配置 AI API Key 才能使用
3. 对话管理接口需要 MongoDB 支持
4. 认证相关接口需要 MongoDB 支持
5. 用户注册后状态为 `inactive`，需要管理员审核激活

## 示例请求

### 用户注册

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "editor001",
    "email": "editor@example.com",
    "password": "123456",
    "nickname": "编辑小王"
  }'
```

**响应**:
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

### 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "editor001",
    "password": "123456"
  }'
```

**响应**:
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

### 获取当前用户信息

```bash
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer {access_token}"
```

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
    }
  }
}
```

### 刷新Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

### 对话接口

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {access_token}" \
  -d '{
    "message": "你好",
    "conversation_id": "conv_123"
  }'
```

### 创建对话

```bash
curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {access_token}" \
  -d '{
    "user_id": "user_123",
    "title": "我的对话",
    "model": "gpt-4"
  }'
```
