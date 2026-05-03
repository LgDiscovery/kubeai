# 生产级 Makefile for KubeAI 平台
# 基础配置
NAMESPACE ?= kubeai
RELEASE_NAME ?= kubeai
KUBECTL ?= kubectl
HELM ?= helm
YQ ?= yq
SED ?= sed
ENV ?= prod
DEPLOY_DIR ?= ./deploy

# 组件列表
COMPONENTS = etcd postgres redis minio prometheus loki
SERVICES = api-gateway model-manager job-scheduler inference-gateway

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
help: ## 显示帮助信息
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ 环境初始化
create-namespace: ## 创建kubeai命名空间
	$(KUBECTL) create namespace $(NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -

install-crd: create-namespace  ## 安装自定义资源 CRD
	@echo "📜 安装自定义资源 CRD..."
	$(KUBECTL) apply -f $(DEPLOY_DIR)/crds/

uninstall-crd:  ## 卸载自定义资源 CRD
	@echo "🗑️ 卸载 CRD..."
	$(KUBECTL) delete -f $(DEPLOY_DIR)/crds/ --ignore-not-found

deploy-configs: create-namespace ## 部署ConfigMap + Secret
	$(KUBECTL) apply -f $(DEPLOY_DIR)/secrets/ -n $(NAMESPACE)
	$(KUBECTL) apply -f $(DEPLOY_DIR)/configs/ -n $(NAMESPACE)

##@ 依赖组件部署
deploy-deps: create-namespace deploy-etcd deploy-postgres deploy-redis deploy-minio deploy-prometheus deploy-loki ## 部署所有依赖组件

deploy-etcd: ## 部署ETCD集群
	$(KUBECTL) apply -f ./deploy/etcd.yaml -n $(NAMESPACE)
	@echo "等待ETCD就绪..."
	$(KUBECTL) wait --for=condition=ready pod -l app=etcd -n $(NAMESPACE) --timeout=5m

deploy-postgres: ## 部署PostgreSQL (Helm)
	$(HELM) upgrade --install postgres bitnami/postgresql -n $(NAMESPACE) -f ./deploy/postgres-values.yaml --create-namespace
	@echo "等待PostgreSQL就绪..."
	$(KUBECTL) wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n $(NAMESPACE) --timeout=5m

deploy-redis: ## 部署Redis集群 (Helm)
	$(HELM) upgrade --install redis bitnami/redis-cluster -n $(NAMESPACE) -f ./deploy/redis-values.yaml --create-namespace
	@echo "等待Redis集群就绪..."
	$(KUBECTL) wait --for=condition=ready pod -l app.kubernetes.io/name=redis-cluster -n $(NAMESPACE) --timeout=10m

deploy-minio: ## 部署MinIO (Helm)
	$(HELM) upgrade --install minio bitnami/minio -n $(NAMESPACE) -f ./deploy/minio-values.yaml --create-namespace
	@echo "创建MinIO models桶..."
	sleep 10
	$(KUBECTL) exec -n $(NAMESPACE) deploy/minio -- mc alias set minio http://localhost:9000 minioadmin minioadmin
	$(KUBECTL) exec -n $(NAMESPACE) deploy/minio -- mc mb minio/models || true

deploy-prometheus: ## 部署Prometheus (Helm)
	$(HELM) repo add prometheus-community https://prometheus-community.github.io/helm-charts
	$(HELM) repo update
	$(HELM) upgrade --install prometheus prometheus-community/prometheus -n $(NAMESPACE) -f ./deploy/prometheus-values.yaml --create-namespace

deploy-loki: ## 部署Loki (Helm)
	$(HELM) repo add grafana https://grafana.github.io/helm-charts
	$(HELM) repo update
	$(HELM) upgrade --install loki grafana/loki -n $(NAMESPACE) -f ./deploy/loki-values.yaml --create-namespace
	$(HELM) upgrade --install promtail grafana/promtail -n $(NAMESPACE) -f ./deploy/promtail-values.yaml --create-namespace

##@ 业务服务部署
deploy-services: deploy-deps install-crd ## 部署所有业务服务
	@for service in $(SERVICES); do \
		echo "部署$$service..."; \
		$(KUBECTL) apply -f ./deploy/server/$$service.yaml -n $(NAMESPACE); \
	done
	@echo "等待所有服务就绪..."
	$(KUBECTL) wait --for=condition=ready pod -l app.kubernetes.io/part-of=kubeai -n $(NAMESPACE) --timeout=5m

##@ 一键全量部署（你要的新功能）
full-deploy: create-namespace install-crd deploy-deps deploy-services ## 🔥 一键全量部署：命名空间+CRD+中间件+业务服务
	@echo ""
	@echo "========================================"
	@echo "✅ KubeAI 全量部署完成！"
	@echo "========================================"
	@echo ""

##@ 一键全量卸载（你要的新功能）
full-undeploy: clean-services clean-deps uninstall-crd ## 🗑️ 一键全量卸载：服务+中间件+CRD+命名空间
	$(KUBECTL) delete namespace $(NAMESPACE) --ignore-not-found
	@echo ""
	@echo "========================================"
	@echo "❌ KubeAI 全量卸载完成！"
	@echo "========================================"
	@echo ""

##@ 配置管理
update-configs: ## 更新服务配置（适配环境）
	@for service in $(SERVICES); do \
		$(YQ) eval ".Etcd.Hosts[0] = \"etcd.$(NAMESPACE).svc.cluster.local:2379\"" ./$$service.yaml > ./$$service.tmp.yaml; \
		$(YQ) eval ".Database.Host = \"postgres.$(NAMESPACE).svc.cluster.local\"" ./$$service.tmp.yaml > ./$$service.yaml; \
		$(YQ) eval ".Redis.Host = \"redis-redis-cluster-0.redis-redis-cluster-headless.$(NAMESPACE).svc.cluster.local:6379,redis-redis-cluster-1.redis-redis-cluster-headless.$(NAMESPACE).svc.cluster.local:6379,redis-redis-cluster-2.redis-redis-cluster-headless.$(NAMESPACE).svc.cluster.local:6379\"" ./$$service.yaml > ./$$service.tmp.yaml; \
		mv ./$$service.tmp.yaml ./$$service.yaml; \
	done
	@echo "配置已更新为生产环境"

##@ 运维操作
logs: ## 查看服务日志 (示例: make logs SERVICE=gateway)
	$(KUBECTL) logs -f -l app=$(SERVICE) -n $(NAMESPACE)

status: ## 查看所有组件状态
	$(KUBECTL) get pods,svc,statefulset -n $(NAMESPACE)

scale: ## 扩缩容服务 (示例: make scale SERVICE=gateway REPLICAS=3)
	$(KUBECTL) scale deploy/$(SERVICE) --replicas=$(REPLICAS) -n $(NAMESPACE)

rollback: ## 回滚服务 (示例: make rollback SERVICE=gateway)
	$(KUBECTL) rollout undo deploy/$(SERVICE) -n $(NAMESPACE)

##@ 清理操作
clean-services: ## 删除所有业务服务
	@for service in $(SERVICES); do \
		$(KUBECTL) delete -f ./deploy/server/$$service.yaml -n $(NAMESPACE) --ignore-not-found; \
	done

clean-deps: ## 删除所有依赖组件
	$(HELM) uninstall postgres redis minio prometheus loki -n $(NAMESPACE) --ignore-not-found
	$(KUBECTL) delete -f ./deploy/etcd.yaml -n $(NAMESPACE) --ignore-not-found

##@ 健康检查
health-check: ## 检查所有组件健康状态
	@echo "=== ETCD 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) statefulset/etcd -- etcdctl endpoint health --endpoints=http://localhost:2379
	@echo "=== PostgreSQL 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) deploy/postgres -- psql -U postgres -d kubeai_platform -c "SELECT 1;"
	@echo "=== Redis 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) pod/redis-redis-cluster-0 -- redis-cli -u redis://redis:redis123@localhost:6379 CLUSTER INFO
	@echo "=== MinIO 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) deploy/minio -- mc admin info minio
	@echo "=== 业务服务健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) deploy/api-gateway -- curl -f http://localhost:8080/api/v1/auth/health 2>/dev/null || echo "api-gateway 未就绪"
	$(KUBECTL) exec -n $(NAMESPACE) deploy/model-manager -- curl -f http://localhost:58080/api/v1/models/health 2>/dev/null || echo "model-manager 未就绪"
	$(KUBECTL) exec -n $(NAMESPACE) deploy/job-scheduler -- curl -f http://localhost:58081/api/v1/jobs/health 2>/dev/null || echo "job-scheduler 未就绪"
	$(KUBECTL) exec -n $(NAMESPACE) deploy/inference-gateway -- curl -f http://localhost:58082/api/v1/inference/health 2>/dev/null || echo "inference-gateway 未就绪"