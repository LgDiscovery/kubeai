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

# 控制器配置目录路径
INFERENCE_CONFIG := $(DEPLOY_DIR)/inferenceService/config
TRAINING_CONFIG := $(DEPLOY_DIR)/trainingJob/config

# 组件列表（按部署yaml对齐）
COMPONENTS = etcd postgres redis minio prometheus loki
# 服务列表（与deploy/server下的yaml文件名前缀对齐）
SERVICES = api-gateway model-manager job-scheduler inference-gateway

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
help: ## 显示帮助信息
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ 环境初始化
create-namespace: ## 创建kubeai命名空间
	$(KUBECTL) create namespace $(NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -

# install-crd 已集成控制器的 CRD 定义 + RBAC + 部署
install-crd: create-namespace  ## 安装CRD定义 + 控制器RBAC + 控制器部署
	@echo "📜 安装核心 CRD 定义..."
	$(KUBECTL) apply -f $(DEPLOY_DIR)/crds/

	@echo "🔧 安装 InferenceService 控制器 RBAC 与部署..."
	# 递归安装 rbac/ 和 manager/ 下的所有 yaml（自动忽略无关文件）
	find $(INFERENCE_CONFIG)/rbac -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)
	find $(INFERENCE_CONFIG)/manager -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)

	@echo "🔧 安装 TrainingJob 控制器 RBAC 与部署..."
	find $(TRAINING_CONFIG)/rbac -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)
	find $(TRAINING_CONFIG)/manager -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)

# uninstall-crd 同步卸载控制器资源（先删部署再删CRD，避免残留）
uninstall-crd:  ## 卸载CRD定义 + 控制器部署 + RBAC
	@echo "🗑️ 卸载控制器部署..."
	find $(INFERENCE_CONFIG)/manager -type f -name '*.yaml' | xargs -I {} $(KUBECTL) delete -f {} -n $(NAMESPACE) --ignore-not-found
	find $(TRAINING_CONFIG)/manager -type f -name '*.yaml' | xargs -I {} $(KUBECTL) delete -f {} -n $(NAMESPACE) --ignore-not-found

	@echo "🗑️ 卸载控制器 RBAC 权限..."
	find $(INFERENCE_CONFIG)/rbac -type f -name '*.yaml' | xargs -I {} $(KUBECTL) delete -f {} -n $(NAMESPACE) --ignore-not-found
	find $(TRAINING_CONFIG)/rbac -type f -name '*.yaml' | xargs -I {} $(KUBECTL) delete -f {} -n $(NAMESPACE) --ignore-not-found

	@echo "🗑️ 卸载核心 CRD 定义..."
	$(KUBECTL) delete -f $(DEPLOY_DIR)/crds/ --ignore-not-found

# 【可选扩展】：安装控制器的网络策略 + Prometheus 监控（按需开启）
install-crd-optional: install-crd ## 安装CRD可选资源（网络策略 + 监控）
	@echo "🔧 安装 InferenceService 网络策略与监控..."
	find $(INFERENCE_CONFIG)/network-policy -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)
	find $(INFERENCE_CONFIG)/prometheus -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)

	@echo "🔧 安装 TrainingJob 网络策略与监控..."
	find $(TRAINING_CONFIG)/network-policy -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)
	find $(TRAINING_CONFIG)/prometheus -type f -name '*.yaml' | xargs -I {} $(KUBECTL) apply -f {} -n $(NAMESPACE)

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
deploy-services: deploy-deps install-crd ## 部署所有业务服务（已依赖install-crd，确保控制器先就绪）
	@for service in $(SERVICES); do \
		echo "部署$$service..."; \
		# 适配deploy/server下的文件名：xxx-service.yaml
		$(KUBECTL) apply -f ./deploy/server/$$service-service.yaml -n $(NAMESPACE); \
	done
	@echo "等待所有服务就绪..."
	# 适配服务的label：app=xxx（与deployment的label对齐）
	$(KUBECTL) wait --for=condition=ready pod -l app in ($(subst $(space),$(comma),$(SERVICES))) -n $(NAMESPACE) --timeout=5m

##@ 一键全量部署
full-deploy: create-namespace install-crd deploy-deps deploy-services ## 🔥 一键全量部署：命名空间+CRD+中间件+业务服务
	@echo ""
	@echo "========================================"
	@echo "✅ KubeAI 全量部署完成！"
	@echo "========================================"
	@echo ""
	@echo "📌 服务访问入口信息："
	@echo "----------------------------------------"
	@echo "🔹 API网关（对外暴露）："
	@echo "   NodePort访问：http://<任意节点IP>:38080"
	@echo "   ClusterIP访问：http://kubeai-gateway.$(NAMESPACE).svc.cluster.local:8080"
	@echo "----------------------------------------"
	@echo "🔹 模型管理服务（集群内访问）："
	@echo "   http://model-manager.$(NAMESPACE).svc.cluster.local:58080"
	@echo "----------------------------------------"
	@echo "🔹 任务调度服务（集群内访问）："
	@echo "   http://job-scheduler.$(NAMESPACE).svc.cluster.local:58081"
	@echo "----------------------------------------"
	@echo "🔹 推理网关服务（集群内访问）："
	@echo "   http://inference-gateway.$(NAMESPACE).svc.cluster.local:58082"
	@echo "----------------------------------------"
	@echo "🔹 监控指标（各服务）："
	@echo "   - API网关：/api/v1/auth/metrics (8080端口)"
	@echo "   - 模型管理：/api/v1/models/metrics (58080端口)"
	@echo "   - 任务调度：/api/v1/job/metrics (58081端口)"
	@echo "   - 推理网关：/api/v1/inference/metrics (58082端口)"
	@echo ""

##@ 一键全量卸载
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
		CONFIG_FILE=$(DEPLOY_DIR)/configs/$$service.yaml; \
		if [ -f $$CONFIG_FILE ]; then \
			$(YQ) eval ".Etcd.Hosts[0] = \"etcd.$(NAMESPACE).svc.cluster.local:2379\"" $$CONFIG_FILE > $$CONFIG_FILE.tmp; \
			$(YQ) eval ".Database.Host = \"postgres.$(NAMESPACE).svc.cluster.local\"" $$CONFIG_FILE.tmp > $$CONFIG_FILE; \
			$(YQ) eval ".Redis.Host = \"redis-redis-cluster-0.redis-redis-cluster-headless.$(NAMESPACE).svc.cluster.local:6379,redis-redis-cluster-1.redis-redis-cluster-headless.$(NAMESPACE).svc.cluster.local:6379,redis-redis-cluster-2.redis-redis-cluster-headless.$(NAMESPACE).svc.cluster.local:6379\"" $$CONFIG_FILE > $$CONFIG_FILE.tmp; \
			mv $$CONFIG_FILE.tmp $$CONFIG_FILE; \
			echo "✅ 更新$$service配置完成"; \
		else \
			echo "⚠️  $$CONFIG_FILE 不存在，跳过"; \
		fi; \
	done
	@echo "📌 配置已更新为生产环境"

##@ 运维操作
logs: ## 查看服务日志 (示例: make logs SERVICE=api-gateway)
	$(KUBECTL) logs -f -l app=$(SERVICE) -n $(NAMESPACE)

status: ## 查看所有组件状态
	$(KUBECTL) get pods,svc,statefulset,deploy -n $(NAMESPACE)

scale: ## 扩缩容服务 (示例: make scale SERVICE=api-gateway REPLICAS=3)
	$(KUBECTL) scale deploy/$(SERVICE) --replicas=$(REPLICAS) -n $(NAMESPACE)

rollback: ## 回滚服务 (示例: make rollback SERVICE=api-gateway)
	$(KUBECTL) rollout undo deploy/$(SERVICE) -n $(NAMESPACE)

##@ 清理操作
clean-services: ## 删除所有业务服务
	@for service in $(SERVICES); do \
		$(KUBECTL) delete -f ./deploy/server/$$service-service.yaml -n $(NAMESPACE) --ignore-not-found; \
	done

clean-deps: ## 删除所有依赖组件
	$(HELM) uninstall postgres redis minio prometheus loki promtail -n $(NAMESPACE) --ignore-not-found
	$(KUBECTL) delete -f ./deploy/etcd.yaml -n $(NAMESPACE) --ignore-not-found

##@ 健康检查
health-check: ## 检查所有组件健康状态
	@echo "=== ETCD 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) statefulset/etcd -- etcdctl endpoint health --endpoints=http://localhost:2379 || echo "❌ ETCD 未就绪"
	@echo "=== PostgreSQL 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) deploy/postgres -- psql -U postgres -d kubeai_platform -c "SELECT 1;" || echo "❌ PostgreSQL 未就绪"
	@echo "=== Redis 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) pod/redis-redis-cluster-0 -- redis-cli -u redis://redis:redis123@localhost:6379 CLUSTER INFO || echo "❌ Redis 未就绪"
	@echo "=== MinIO 健康检查 ==="
	$(KUBECTL) exec -n $(NAMESPACE) deploy/minio -- mc admin info minio || echo "❌ MinIO 未就绪"
	@echo "=== 业务服务健康检查 ==="
	@echo "🔹 API网关："
	$(KUBECTL) exec -n $(NAMESPACE) deploy/api-gateway -- curl -f http://localhost:8080/api/v1/auth/metrics 2>/dev/null && echo "✅ api-gateway 就绪" || echo "❌ api-gateway 未就绪"
	@echo "🔹 模型管理服务："
	$(KUBECTL) exec -n $(NAMESPACE) deploy/model-manager -- curl -f http://localhost:58080/api/v1/models/metrics 2>/dev/null && echo "✅ model-manager 就绪" || echo "❌ model-manager 未就绪"
	@echo "🔹 任务调度服务："
	$(KUBECTL) exec -n $(NAMESPACE) deploy/job-scheduler -- curl -f http://localhost:58081/api/v1/jobs/metrics 2>/dev/null && echo "✅ job-scheduler 就绪" || echo "❌ job-scheduler 未就绪"
	@echo "🔹 推理网关服务："
	$(KUBECTL) exec -n $(NAMESPACE) deploy/inference-gateway -- curl -f http://localhost:58082/api/v1/inference/metrics 2>/dev/null && echo "✅ inference-gateway 就绪" || echo "❌ inference-gateway 未就绪"