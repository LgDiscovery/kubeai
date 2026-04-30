## 1 依赖组件一键部署

### 1.1 创建命名空间

```shell
kubectl create namespace kubeai
```

### 1.2 部署PostgreSQL

```shell
helm install postgres oci://registry-1.docker.io/bitnamicharts/postgresql \
  --namespace kubeai \
  --set auth.postgresPassword=postgres \
  --set auth.database=kubeai_platform
```

### 1.3 部署Minio 模型仓库

```shell
helm install minio oci://registry-1.docker.io/bitnamicharts/minio \
  --namespace kubeai \
  --set rootUser=minioadmin \
  --set rootPassword=minioadmin \
  --set defaultBucket.enabled=true \
  --set defaultBucket.name=models
```

### 1.4 部署Redis 任务队列、缓存 分布式锁

```shell
helm install redis oci://registry-1.docker.io/bitnamicharts/redis-cluster \
  --namespace kubeai \
  --set auth.password=redis123
```

### 1.5 部署可观测套件

```shell
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
# 部署Prometheus
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace kubeai \
  --set prometheus.prometheusSpec.serviceMonitorSelector.matchLabels.kubeai=monitoring
```

### 1.6 部署Loki+Promtail

```shell
helm install loki grafana/loki-stack --namespace kubeai
```

## 2 平台服务部署

### 2.1 部署 CRD

```
make install  # 安装InferenceService/TrainingJob CRD到集群
```

### 2.2 部署核心服务

```
# 部署模型管理服务
kubectl apply -f deployments/manifests/model-manager.yaml -n kubeai
# 部署任务调度中心
kubectl apply -f deployments/manifests/job-scheduler.yaml -n kubeai
# 部署推理Operator
kubectl apply -f deployments/manifests/inference-gateway.yaml -n kubeai
# 部署API网关
kubectl apply -f deployments/gateway/api-gateway.yaml -n kubeai
```

## 3 配置服务间通信

### 所有服务通过 K8s Service 名称进行内网通信，无需公网暴露，核心服务域名：

* 模型管理中心：`http://model-manager.kubeai.svc.cluster.local:8080`
* 任务调度中心：`http://job-scheduler.kubeai.svc.cluster.local:8081`
* 推理网关：`http://inference-gateway.kubeai.svc.cluster.local:8082`
* MinIO：`http://minio.kubeai.svc.cluster.local:9000`
* PostgreSQL：`postgres://postgres:postgres@postgres.kubeai.svc.cluster.local:5432/ai_platform`
* Redis：`redis://:redis123@redis.kubeai.svc.cluster.local:6379`
