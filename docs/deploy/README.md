# 部署文档

## 目录说明

本目录包含 Lemon 后端项目的部署相关文档。

## 文档列表

### Docker 部署
- Dockerfile 配置说明
- docker-compose 使用指南
- 镜像构建和推送

### Kubernetes 部署
- K8s 配置文件
- 服务部署指南
- 配置管理
- 扩缩容配置

### 环境配置
- 开发环境配置
- 测试环境配置
- 生产环境配置

### 监控和日志
- 日志收集配置
- 监控指标配置
- 告警配置

## 快速开始

### Docker 部署

```bash
# 构建镜像
docker build -t lemon:latest -f deployments/Dockerfile .

# 运行容器
docker-compose -f deployments/docker-compose.yaml up -d
```

### Kubernetes 部署

```bash
# 应用配置
kubectl apply -f deployments/k8s/

# 查看状态
kubectl get pods -n lemon
```

## 相关文件

- `deployments/Dockerfile` - Docker 镜像构建文件
- `deployments/docker-compose.yaml` - Docker Compose 配置
- `deployments/k8s/` - Kubernetes 配置文件（如有）

## 注意事项

- 生产环境部署前，请确保已配置所有必要的环境变量
- 数据库和Redis连接配置需要根据实际环境调整
- 建议使用配置管理工具（如 ConfigMap、Secret）管理敏感信息
