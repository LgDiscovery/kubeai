# KubeAI 部署指南

## 目录

- [架构概述](#架构概述)
- [系统要求](#系统要求)
- [快速开始（Docker Compose）](#快速开始docker-compose)
- [Kubernetes 部署](#kubernetes-部署)
- [配置说明](#配置说明)
- [验证部署](#验证部署)
- [常见问题](#常见问题)

---

## 架构概述

KubeAI 是一个云原生 AI 服务平台，采用微服务架构：

```
┌─────────────────────────────────────────────────────────────┐
│                     前端 (kubeai-frontend)                   │
│         React 18 + TypeScript + Vite + Ant Design 5         │
└──────────────────────────────┬──────────────────────────────┘
                               │ HTTP
┌──────────────────────────────▼──────────────────────────────┐
│                    API 网关 (api-gateway :8080)              │
│              登录/注册 · JWT鉴权 · 请求代理 · 指标            │
└──────────────┬───────────────────┬───────────────────────────┘
               │                   │                           │
    ┌──────────▼──────┐  ┌────────▼─────────┐  ┌────────────▼──────────┐
    │ 模型管理服务      │  │ 任务调度服务      │  │ 推理网关服务           │
    │ model-manager    │  │ job-scheduler    │  │ inference-gateway     │
    │ :58080           │  │ :58081           │  │ :58082                │
    │                  │  │                  │  │                       │
    │ • 模型CRUD       │  │ • 训练任务提交    │  │ • 推理执行            │
    │ • 版本管理       │  │ • 推理任务提交    │  │ • 任务控制(暂停/恢复) │
    │ • MinIO存储      │  │ • Redis队列       │  │ • K8s Operator        │
    │ • 元数据管理     │  │ • 幂等性控制      │  │   - InferenceService  │
    │                  │  │ • 状态回调        │  │   - TrainingJob       │
    └──────────────────┘  └──────────────────┘  │   - 灰度发布/HPA      │
                                                  └───────────────────────┘
```

---

## 系统要求

### 最低配置
- CPU: 4 核
- 内存: 8 GB
- 磁盘: 50 GB
- 操作系统: Linux / macOS / Windows

### 推荐配置（生产环境）
- CPU: 16 核+
- 内存: 32 GB+
- 磁盘: 200 GB+ SSD
- Kubernetes: 1.24+
- GPU: NVIDIA GPU（可选，用于模型推理）

### 软件依赖
- Docker 20.10+
- Docker Compose 2.0+（快速部署）
- Kubernetes 1.24+（生产部署）
- kubectl 1.24+
- Helm 3.0+（可选）

---

## 快速开始（Docker Compose）

### 1. 克隆仓库

```bash
git clone https://github.com/LgDiscovery/kubeai.git
cd kubeai
```

### 2. 配置环境变量

复制示例配置文件：

```bash
cp .env.example .env
```

编辑 `.env` 文件，配置必要的参数：

```env
# 数据库配置
POSTGRES_USER=kubeai
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=kubeai

# Redis 配置
REDIS_PASSWORD=your_redis_password

# MinIO 配置
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=your_minio_secret_key
MINIO_BUCKET=ai-models

# JWT 配置
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRE_HOURS=24

# 服务端口
API_GATEWAY_PORT=8080
MODEL_MANAGER_PORT=58080
JOB_SCHEDULER_PORT=58081
INFERENCE_GATEWAY_PORT=58082
```

### 3. 启动服务

```bash
docker-compose up -d
```

### 4. 查看服务状态

```bash
docker-compose ps
```

### 5. 访问前端

打开浏览器访问：`http://localhost:3000`

默认管理员账号：
- 用户名: `admin`
- 密码: `admin123`

### 6. 停止服务

```bash
docker-compose down
```

---

## Kubernetes 部署

### 1. 准备 Kubernetes 集群

确保你有一个可用的 Kubernetes 集群，并且 kubectl 已配置好。

```bash
kubectl cluster-info
```

### 2. 创建命名空间

```bash
kubectl create namespace kubeai
```

### 3. 安装 CRD

```bash
kubectl apply -f deploy/crds/
```

这将安装以下 CRD：
- `inferenceservices.kubeai.io` - 推理服务
- `trainingjobs.kubeai.io` - 训练任务

### 4. 配置 Secret

创建数据库、Redis、MinIO 的 Secret：

```bash
kubectl create secret generic kubeai-secrets \
  --namespace kubeai \
  --from-literal=postgres-password='your_password' \
  --from-literal=redis-password='your_password' \
  --from-literal=minio-access-key='minioadmin' \
  --from-literal=minio-secret-key='your_secret_key' \
  --from-literal=jwt-secret='your_jwt_secret'
```

### 5. 部署基础设施

```bash
kubectl apply -f deploy/infrastructure/
```

这将部署：
- PostgreSQL
- Redis Cluster
- MinIO

### 6. 部署微服务

```bash
kubectl apply -f deploy/services/
```

这将部署：
- api-gateway
- model-manager
- job-scheduler
- inference-gateway（包含 Operator）

### 7. 部署前端

```bash
kubectl apply -f deploy/frontend/
```

### 8. 验证部署

```bash
kubectl get pods -n kubeai
kubectl get svc -n kubeai
```

### 9. 访问服务

获取前端服务地址：

```bash
kubectl get svc kubeai-frontend -n kubeai
```

---

## 配置说明

### 环境变量

#### 通用配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | 日志格式 (json/text) | `json` |

#### 数据库配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DB_HOST` | 数据库主机 | `postgres` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_USER` | 数据库用户 | `kubeai` |
| `DB_PASSWORD` | 数据库密码 | - |
| `DB_NAME` | 数据库名 | `kubeai` |
| `DB_SSL_MODE` | SSL 模式 | `disable` |
| `DB_MAX_IDLE_CONNS` | 最大空闲连接数 | `10` |
| `DB_MAX_OPEN_CONNS` | 最大打开连接数 | `100` |

#### Redis 配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `REDIS_HOST` | Redis 主机 | `redis` |
| `REDIS_PORT` | Redis 端口 | `6379` |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `REDIS_DB` | Redis 数据库 | `0` |

#### MinIO 配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `MINIO_ENDPOINT` | MinIO 地址 | `minio:9000` |
| `MINIO_ACCESS_KEY` | Access Key | - |
| `MINIO_SECRET_KEY` | Secret Key | - |
| `MINIO_BUCKET` | 存储桶名 | `ai-models` |
| `MINIO_USE_SSL` | 是否使用 SSL | `false` |

#### JWT 配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `JWT_SECRET` | JWT 密钥 | - |
| `JWT_EXPIRE_HOURS` | Token 过期时间（小时） | `24` |

---

## 验证部署

### 1. 健康检查

```bash
# API 网关
curl http://localhost:8080/api/v1/auth/health

# 模型管理服务
curl http://localhost:58080/api/v1/model/health

# 任务调度服务
curl http://localhost:58081/api/v1/job/health

# 推理网关服务
curl http://localhost:58082/api/v1/inference/health
```

### 2. 就绪检查

```bash
curl http://localhost:8080/api/v1/auth/ready
```

### 3. 指标端点

```bash
curl http://localhost:8080/api/v1/metrics
```

### 4. 登录测试

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

---

## 常见问题

### Q: 服务启动失败，提示数据库连接错误

A: 检查数据库是否正常运行，确认环境变量 `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD` 配置正确。

### Q: MinIO 上传模型文件失败

A: 检查 MinIO 服务是否正常运行，确认 `MINIO_ENDPOINT`、`MINIO_ACCESS_KEY`、`MINIO_SECRET_KEY` 配置正确，确认存储桶已创建。

### Q: 推理服务创建后状态一直是 Pending

A: 检查 Kubernetes 集群是否有足够的资源（CPU/内存/GPU），查看 Pod 事件：`kubectl describe pod <pod-name> -n kubeai`

### Q: 如何查看服务日志

```bash
# 查看单个服务日志
kubectl logs -f deployment/api-gateway -n kubeai

# 查看所有服务日志
kubectl logs -f -l app=kubeai -n kubeai
```

### Q: 如何升级服务

```bash
# 更新镜像
kubectl set image deployment/api-gateway api-gateway=new-image:tag -n kubeai

# 查看滚动更新状态
kubectl rollout status deployment/api-gateway -n kubeai
```

---

## 相关文档

- [API 文档](./API.md)
- [架构设计](./ARCHITECTURE.md)
- [开发指南](./DEVELOPMENT.md)
- [运维手册](./OPERATIONS.md)
