# 开发规范文档

## 📋 开发流程规范

### 1. 开发前必读设计文档

**重要规则**：在进行任何模块的开发任务前，**必须先查看** `docs/design/` 目录下对应模块的设计文档。

#### 1.1 设计文档位置

所有模块的设计文档都位于 `docs/design/` 目录下，按模块分类：

```
docs/
├── design/                 # 设计文档目录
│   ├── auth/               # 认证模块
│   │   └── AUTH_DESIGN.md  # 认证系统设计文档
│   └── workflow/           # 工作流模块
│       ├── ARCHITECTURE.md # 工作流架构设计
│       └── API_DESIGN.md   # 工作流API设计
└── deploy/                 # 部署文档目录
    └── README.md           # 部署文档索引
```

#### 1.2 开发流程

1. **接收开发任务**
   - 明确要开发的模块（如：认证模块、工作流模块）

2. **查阅设计文档** ⚠️ **必须步骤**
   - 进入 `docs/design/{模块名}/` 目录
   - 阅读相关的设计文档
   - 理解：
     - 数据模型设计
     - API接口设计
     - 业务逻辑设计
     - 权限控制设计
     - 错误处理设计

3. **确认理解**
   - 确保理解设计文档中的所有内容
   - 如有疑问，先与架构师/技术负责人确认
   - 不要在没有理解设计的情况下开始编码

4. **开始开发**
   - 严格按照设计文档进行开发
   - 如有设计变更需求，先更新设计文档，再修改代码

#### 1.3 各模块对应的设计文档

| 模块 | 设计文档路径 | 主要内容 |
|------|------------|---------|
| 认证模块 | `docs/design/auth/AUTH_DESIGN.md` | 用户认证、JWT机制、角色权限、用户管理API |
| 工作流模块 | `docs/design/workflow/ARCHITECTURE.md` | 系统架构、数据模型、任务队列、文件存储 |
| 工作流API | `docs/design/workflow/API_DESIGN.md` | 完整的API接口定义、请求响应格式 |

### 2. 代码规范

#### 2.1 文件组织

- 遵循 Clean Architecture 分层架构
- 文件按模块组织，与设计文档中的模块划分保持一致

#### 2.1.1 Handler层文件组织 ⚠️ **重要规范**

**Request/Response DTO位置**:
- ✅ **Request/Response DTO必须定义在Handler层**，不能放在Model层
- ✅ **每个API一个文件**，文件命名：`{module}/{api_name}.go`
- ✅ 每个API文件包含：
  - 该API的Request结构体定义
  - 该API的Response结构体定义
  - 该API的Handler方法
- ✅ 共用的类型（如ErrorResponse、UserInfo）放在`{module}/common.go`中

**文件组织示例**:
```
internal/handler/auth/
├── handler.go          # Handler结构体定义
├── common.go           # 共用的ErrorResponse、UserInfo等
├── register.go         # 注册API（RegisterRequest + RegisterResponse + Register方法）
├── login.go            # 登录API（LoginRequest + LoginResponse + Login方法）
├── refresh.go          # 刷新Token API（RefreshTokenRequest + RefreshTokenResponse + Refresh方法）
├── logout.go           # 退出登录API（Logout方法）
└── me.go               # 获取当前用户信息API（GetMe方法）
```

**Service层规范**:
- ✅ Service层使用基本类型参数（如`string`, `int`等），不依赖Handler层的Request/Response类型
- ✅ Service层返回内部结果类型（如`RegisterResult`, `LoginResult`），由Handler层转换为Response DTO

**示例**:
```go
// ✅ 正确：Handler层定义Request/Response
// internal/handler/auth/register.go
package auth

type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}

type RegisterResponseData struct {
    UserID   string `json:"user_id"`
    Username string `json:"username"`
    Status   string `json:"status"`
}

func (h *Handler) Register(c *gin.Context) {
    var req RegisterRequest
    // 调用Service层，传递基本类型参数
    resp, err := h.authService.Register(ctx, req.Username, req.Email, req.Password, req.Nickname)
    // 转换为Response DTO
    c.JSON(http.StatusCreated, RegisterResponseData{...})
}
```

```go
// ✅ 正确：Service层使用基本类型参数
// internal/service/auth_service.go
type RegisterResult struct {
    UserID   string
    Username string
    Status   string
}

func (s *AuthService) Register(ctx context.Context, username, email, pwd, nickname string) (*RegisterResult, error) {
    // 业务逻辑，不依赖Handler层的Request类型
}
```

#### 2.2 命名规范

- 包名：小写，使用下划线（如：`internal/handler/auth.go`）
- 结构体：大驼峰（如：`User`、`Workflow`）
- 函数/方法：大驼峰（如：`CreateUser`、`GetWorkflow`）
- 变量：小驼峰（如：`userID`、`workflowStatus`）

#### 2.3 注释规范

- 所有公开的函数、结构体必须有文档注释
- 复杂业务逻辑必须有行内注释
- API接口必须有Swagger注释

### 3. 设计文档更新规范

#### 3.1 何时更新设计文档

- 新增功能模块时，必须先创建设计文档
- 修改现有功能时，如果涉及架构变更，必须先更新设计文档
- 修复bug时，如果发现设计问题，需要更新设计文档

#### 3.2 如何更新设计文档

1. 在 `docs/design/{模块名}/` 目录下找到对应的设计文档
2. 更新相关章节
3. 更新文档的"最后更新日期"
4. 在提交代码时，同时提交设计文档的更新

### 4. API开发规范

#### 4.1 API设计

- 所有API接口必须在设计文档中定义
- 遵循RESTful规范
- 统一的请求/响应格式
- 统一的错误码定义

#### 4.2 参数传递规范 ⚠️ **重要规则**

**严格遵循以下规则**：

1. **不使用路径参数（Path Parameters）**
   - ❌ 禁止使用：`GET /api/v1/users/:id`
   - ✅ 正确使用：`GET /api/v1/users?id=xxx`

2. **GET请求参数**
   - 所有参数必须通过Query String传递
   - 示例：`GET /api/v1/users?id=123&role=editor`

3. **POST/PUT/DELETE请求参数**
   - 所有参数必须通过Request Body传递
   - 包括ID等标识符，也要放在body中
   - 示例：
     ```json
     POST /api/v1/users/update
     {
       "id": "123",
       "role": "editor"
     }
     ```

4. **路由设计示例**

   | 错误示例 | 正确示例 |
   |---------|---------|
   | `GET /api/v1/users/:id` | `GET /api/v1/users?id=xxx` |
   | `PUT /api/v1/users/:id` | `POST /api/v1/users/update` (id在body中) |
   | `DELETE /api/v1/users/:id` | `POST /api/v1/users/delete` (id在body中) |
   | `POST /api/v1/users/:id/approve` | `POST /api/v1/users/approve` (id在body中) |

#### 4.2 API实现

- 严格按照设计文档中的API定义实现
- 请求参数验证
- 错误处理
- 日志记录

### 5. 数据库设计规范

#### 5.1 数据模型

- 所有数据模型必须在设计文档中定义
- 使用MongoDB的BSON标签
- 定义必要的索引

#### 5.2 Repository层

**文件组织**:
- ✅ **按模块组织**，每个模块一个子目录
- ✅ 文件命名：`{module}/{entity}_repo.go`
- ✅ 包名使用模块名（`package auth`, `package resource`等）

**文件组织示例**:
```
internal/repository/
├── auth/
│   ├── user_repo.go          # 用户仓库
│   └── refresh_token_repo.go  # RefreshToken仓库
├── resource/
│   └── resource_repo.go       # 资源仓库
└── ...
```

**规范**:
- 所有数据库操作通过Repository层
- Repository接口定义清晰
- 使用依赖注入，通过构造函数接受数据库连接
- 所有方法接受`context.Context`作为第一个参数
- 错误处理完善

### 6. 测试规范

#### 6.1 单元测试

- 核心业务逻辑必须有单元测试
- 测试覆盖率目标：>= 70%

#### 6.2 集成测试

- API接口必须有集成测试
- 测试用例覆盖正常流程和异常流程

### 7. 提交规范

#### 7.1 Commit Message

遵循 Conventional Commits 规范：

```
feat: 新功能
fix: 修复bug
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试
chore: 构建/工具
```

#### 7.2 提交内容

- 代码变更
- 设计文档更新（如有）
- 测试用例
- 相关配置文件

### 8. 代码审查检查清单

在提交代码审查前，请确认：

- [ ] 已阅读并理解相关模块的设计文档
- [ ] 代码实现符合设计文档
- [ ] 已添加必要的注释
- [ ] 已添加单元测试
- [ ] 已更新相关文档（如有变更）
- [ ] 代码通过lint检查
- [ ] 提交信息符合规范

## 📚 相关文档

- [API使用文档](../api/README.md) - API快速开始
- [架构设计文档](../design/workflow/ARCHITECTURE.md) - 系统架构设计
- [认证系统设计](../design/auth/AUTH_DESIGN.md) - 认证模块设计

## ⚠️ 重要提醒

**开发前不阅读设计文档，可能导致：**
- 实现与设计不一致
- 需要大量返工
- 代码质量问题
- 架构混乱

**请务必在开始编码前，先阅读并理解设计文档！**
