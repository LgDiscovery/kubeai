# 基于Go的AI服务平台

## kubeai 是基于Go语言+kubernetes自研的极简AI平台，三大核心模块形成「模型生命周期→训练调度→在线推理」的完整闭环，定位与交互如下：


| 模块             | 核心定位     | 核心能力                                                                  | 与其他模块交互                                                                     |
| :--------------- | :----------- | :------------------------------------------------------------------------ | :--------------------------------------------------------------------------------- |
| **模型管理服务** | 平台数据底座 | 模型注册 / 版本管理 / 元数据存储 / MinIO 文件管理，替代 MLflow 核心能力   | 为**推理网关**提供部署用的模型文件与元数据；为**任务调度**提供训练后模型的入库入口 |
| **推理服务网关** | 在线服务入口 | 基于 K8s Operator（CRD+Controller）实现模型声明式部署、灰度发布、弹性伸缩 | 从**模型管理**拉取模型；接收外部推理请求；将服务状态同步给平台                     |
| **任务调度服务** | 离线训练中枢 | 基于 K8s Job/TrainingJob CRD 管理训练任务、状态跟踪、日志收集             | 训练完成后将模型上传至**模型管理**；接收算法团队训练任务                           |

***架构***

![KubeAI 架构简图](assets/arch-simple.png)

***核心数据流***

1. **训练流**：提交训练任务→任务调度服务下发任务→redis stream 队列->推理网关消费任务->创建K8s trainingJob 执行训练→生成模型→上传模型管理服务
2. **部署推理流**：提交部署推理服务任务→任务调度服务下发任务->redis stream 队列->推理网关消费任务->从模型管理拉取模型→K8s inferenceservice创建 Deployment/Service/ingresss→对外提供推理 API
3. **推理流**：提交推理任务->推理网关->ingree路由具体实例进行推理→返回推理结果

## 核心功能

### 1. 模型管理服务（替代 MLflow）

* 模型注册 / 版本管理 / 元数据存储
* MinIO 分布式模型文件存储
* 模型状态管理（active/archived）

### 2. 推理服务网关（K8s Operator）

* InferenceService 自定义资源
* 声明式部署、灰度发布、弹性伸缩（HPA）
* 自动生成 Deployment/Service/Ingress/HPA

### 3. 任务调度服务（离线训练）

* TrainingJob 自定义资源
* PyTorch/TensorFlow 训练任务编排
* 任务状态同步、日志实时拉取

## 技术栈

* 后端：Go、Go-zero、GORM、client-go、Kubebuilder
* 存储：PostgreSQL、Redis、MinIO
* 云原生：K8s、CRD/Controller、Job/HPA/Ingress-Nginx
* 可观测：Prometheus、Grafana、ELK
* 部署：Docker、K8s、kubeadm
