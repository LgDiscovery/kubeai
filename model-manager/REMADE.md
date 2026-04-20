1. **配置文件**：`etc/model-manager.yaml` 中的数据库、MinIO 地址需与您的 K8s 环境一致。
2. **指标包**：`pkg/metrics` 已在之前回答中提供，确保引入路径正确。
3. **网关转发**：API 网关需将 `/api/v1/models/*` 路由转发至 `http://model-manager.kubeai.svc.cluster.local:58080`。
4. **JWT 鉴权**：模型管理服务自身不验证 JWT，由网关统一处理，从请求头获取用户信息（需网关透传）。
