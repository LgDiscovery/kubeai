# KubeAI API 文档

## 概述

KubeAI 提供 RESTful API，所有 API 都遵循统一的响应格式。

**Base URL**: `http://<host>:<port>/api/v1`

**认证方式**: Bearer Token (JWT)

**内容类型**: `application/json`

---

## 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 状态码，0 表示成功，非 0 表示失败 |
| `message` | string | 状态消息 |
| `data` | object | 响应数据 |

---

## 认证 API

### 登录

**POST** `/auth/login`

请求体：
```json
{
  "username": "admin",
  "password": "admin123"
}
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "1",
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 注册

**POST** `/auth/register`

请求体：
```json
{
  "username": "newuser",
  "password": "password123",
  "email": "user@example.com"
}
```

响应：
```json
{
  "code": 0,
  "message": "注册成功",
  "data": {
    "id": "2",
    "username": "newuser"
  }
}
```

### 健康检查

**GET** `/auth/health`

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "UP",
    "service": "api-gateway",
    "timestamp": "2025-01-01T00:00:00Z"
  }
}
```

### 就绪检查

**GET** `/auth/ready`

响应：
```json
{
  "code": 0,
  "message": "service is ready",
  "data": {
    "status": "READY",
    "service": "api-gateway",
    "timestamp": "2025-01-01T00:00:00Z",
    "dependencies": {
      "database": { "status": "UP", "message": "" },
      "redis": { "status": "UP", "message": "" }
    }
  }
}
```

---

## 模型管理 API

### 列出模型

**GET** `/model/models`

查询参数：
| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页数量，默认 20 |

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 10,
    "items": [
      {
        "id": "model-uuid",
        "name": "bert-sentiment",
        "type": "llm",
        "provider": "open_ai",
        "model_name": "bert-base-uncased",
        "status": "active",
        "created_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

### 创建模型

**POST** `/model/models`

请求体：
```json
{
  "name": "my-model",
  "type": "llm",
  "provider": "open_ai",
  "model_name": "gpt-3.5-turbo",
  "endpoint": "https://api.openai.com/v1",
  "api_key": "sk-xxx",
  "config": {}
}
```

### 获取模型详情

**GET** `/model/models/{id}`

### 删除模型

**DELETE** `/model/models/{id}`

### 创建模型版本（支持文件上传）

**POST** `/model/models/{name}/versions`

**Content-Type**: `multipart/form-data`

表单字段：
| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 模型名称 |
| `version` | string | 版本号，如 v1.0.0 |
| `description` | string | 版本描述 |
| `framework` | string | 框架（PyTorch/TensorFlow/ONNX） |
| `framework_version` | string | 框架版本 |
| `metrics` | string | 评估指标（JSON） |
| `parameters` | string | 超参数（JSON） |
| `file` | file | 模型文件 |

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "version-uuid",
    "model_id": "model-uuid",
    "version": "v1.0.0",
    "storage_path": "models/my-model/v1.0.0/uuid.bin",
    "size": 104857600,
    "checksum": "abc123...",
    "status": "active",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

### 列出模型版本

**GET** `/model/models/{name}/versions`

### 获取版本详情

**GET** `/model/models/{name}/versions/{version}`

### 下载模型文件

**GET** `/model/models/{name}/versions/{version}/download`

响应：文件流

### 健康检查

**GET** `/model/health`

### 就绪检查

**GET** `/model/ready`

---

## 任务调度 API

### 提交训练任务

**POST** `/job/training`

请求体：
```json
{
  "name": "training-task-1",
  "model_name": "bert-sentiment",
  "model_version": "v1.0.0",
  "framework": "pytorch",
  "image": "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime",
  "command": ["python", "train.py"],
  "args": ["--epochs", "10"],
  "resources": {
    "cpu": "4",
    "memory": "16Gi",
    "gpu": "1"
  },
  "env": [
    {"name": "DATASET_PATH", "value": "/data/train"}
  ],
  "priority": "normal",
  "max_retries": 3
}
```

响应：
```json
{
  "code": 0,
  "message": "任务提交成功",
  "data": {
    "task_id": "task-uuid",
    "status": "pending"
  }
}
```

### 提交推理任务

**POST** `/job/inference`

请求体：
```json
{
  "model_name": "bert-sentiment",
  "model_version": "v1.0.0",
  "input": {
    "text": "这是一个测试"
  },
  "parameters": {
    "temperature": 0.7
  },
  "priority": "normal"
}
```

### 列出任务

**GET** `/job/tasks`

查询参数：
| 参数 | 类型 | 说明 |
|------|------|------|
| `task_type` | string | 任务类型（training/inference） |
| `status` | string | 状态筛选 |
| `page` | int | 页码 |
| `page_size` | int | 每页数量 |

### 获取任务详情

**GET** `/job/tasks/{task_id}`

### 取消任务

**POST** `/job/tasks/{task_id}/cancel`

### 任务回调

**POST** `/job/callback`

---

## 推理网关 API

### 执行推理

**POST** `/inference/execute`

请求体：
```json
{
  "model_name": "bert-sentiment",
  "model_version": "v1.0.0",
  "input": {
    "text": "这是一个测试"
  },
  "framework": "pytorch"
}
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task-uuid",
    "result": {
      "label": "positive",
      "score": 0.95
    }
  }
}
```

### 推理服务管理

#### 列出推理服务

**GET** `/inference/services`

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 2,
    "items": [
      {
        "name": "bert-sentiment-v1",
        "model_name": "bert-sentiment",
        "model_version": "v1.0.0",
        "image": "inference-server:latest",
        "replicas": 2,
        "ready_replicas": 2,
        "status": "Running",
        "cpu": "2",
        "memory": "4Gi",
        "gpu": "1",
        "url": "http://bert-sentiment-v1.kubeai.svc.cluster.local",
        "canary_enabled": false,
        "canary_traffic": 0,
        "created_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### 创建推理服务

**POST** `/inference/services`

请求体：
```json
{
  "name": "my-inference-service",
  "model_name": "bert-sentiment",
  "model_version": "v1.0.0",
  "image": "inference-server:latest",
  "replicas": 2,
  "port": 8501,
  "cpu": "2",
  "memory": "4Gi",
  "gpu": "1",
  "canary_enabled": false,
  "canary_traffic": 0,
  "enable_autoscaling": true,
  "max_replicas": 5
}
```

#### 获取推理服务详情

**GET** `/inference/services/{name}`

#### 更新推理服务（扩缩容/镜像更新）

**PATCH** `/inference/services/{name}`

请求体（所有字段可选）：
```json
{
  "replicas": 3,
  "image": "inference-server:v2",
  "cpu": "4",
  "memory": "8Gi",
  "gpu": "2",
  "canary_enabled": true,
  "canary_traffic": 20
}
```

#### 删除推理服务

**DELETE** `/inference/services/{name}`

### 任务控制

#### 取消训练任务

**POST** `/inference/control/tasks/{task_id}/cancel`

#### 暂停训练任务

**POST** `/inference/control/tasks/{task_id}/pause`

#### 恢复训练任务

**POST** `/inference/control/tasks/{task_id}/resume`

#### 重试任务

**POST** `/inference/control/tasks/{task_id}/retry`

### 任务日志

#### 获取任务日志

**GET** `/inference/tasks/{task_id}/logs`

查询参数：
| 参数 | 类型 | 说明 |
|------|------|------|
| `tail_lines` | int | 返回最后 N 行日志 |
| `container` | string | 指定容器名 |
| `since_time` | string | 起始时间（RFC3339 格式） |

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task-uuid",
    "pod_name": "training-task-uuid-0",
    "logs": [
      "[2025-01-01T00:00:00Z] INFO Starting training process...",
      "[2025-01-01T00:00:01Z] INFO Loading dataset..."
    ],
    "timestamp": "2025-01-01T00:00:00Z"
  }
}
```

#### 列出任务 Pod

**GET** `/inference/tasks/{task_id}/pods`

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task-uuid",
    "pods": [
      {
        "name": "training-task-uuid-0",
        "status": "Running",
        "node_name": "node-1",
        "created_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

### 健康检查

**GET** `/inference/health`

### 就绪检查

**GET** `/inference/ready`

---

## 指标 API

### Prometheus 指标

**GET** `/metrics`

返回 Prometheus 格式的指标数据，包括：
- HTTP 请求总数（按方法、路径、状态码）
- HTTP 请求延迟（直方图）
- 模型总数
- 模型版本数
- 任务队列深度
- 推理服务副本数
- 数据库连接池指标

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

---

## 示例

### 使用 curl 调用 API

```bash
# 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

# 列出模型
curl -s http://localhost:8080/api/v1/model/models \
  -H "Authorization: Bearer $TOKEN"

# 创建推理服务
curl -s -X POST http://localhost:8080/api/v1/inference/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-service",
    "model_name": "bert-sentiment",
    "model_version": "v1.0.0",
    "replicas": 1,
    "cpu": "2",
    "memory": "4Gi"
  }'

# 获取任务日志
curl -s "http://localhost:8080/api/v1/inference/tasks/task-id/logs?tail_lines=100" \
  -H "Authorization: Bearer $TOKEN"
```
