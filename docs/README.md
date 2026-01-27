# Lemon 后端项目文档

## 📚 文档导航

### 快速开始
- **[API文档](./api/README.md)** - API快速开始和使用说明
- **[开发规范](./guide/DEVELOPMENT_RULES.md)** - ⚠️ **开发前必读** - 开发流程和规范

### 设计文档
- **[认证模块设计](./design/auth/AUTH_DESIGN.md)** - 认证与权限系统设计
- **[工作流架构设计](./design/workflow/ARCHITECTURE.md)** - 系统架构设计
- **[工作流API设计](./design/workflow/API_DESIGN.md)** - 工作流API接口设计
- **[资产文件管理设计](./design/workflow/ASSET_FILE_DESIGN.md)** - 资产文件上传下载模块设计

### 部署文档
- **[部署文档](./deploy/README.md)** - Docker、K8s部署指南

### Swagger文档
启动服务器后访问：**http://localhost:8080/swagger/index.html**

## 📁 文档结构

```
docs/
├── README.md              # 本文档（文档导航）
├── api/                   # API文档
│   └── README.md          # API使用文档
├── guide/                 # 开发指南
│   └── DEVELOPMENT_RULES.md  # 开发规范
├── design/                # 设计文档
│   ├── auth/              # 认证模块
│   └── workflow/          # 工作流模块
├── deploy/                # 部署文档
└── swagger/               # Swagger自动生成文档
```

## ⚠️ 重要提醒

**开发前必读**：在进行任何模块开发前，请先阅读 [开发规范](./guide/DEVELOPMENT_RULES.md)。
