// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	k8stype "k8s.io/apimachinery/pkg/types"
	aiv1 "kubeai-inference-gateway/inferenceservice/api/v1"
	"kubeai-inference-gateway/internal/model"
	"kubeai-inference-gateway/internal/types"
	"net/http"
	"net/url"
	"time"

	"kubeai-inference-gateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type InferenceLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	httpClient *http.Client
}

func NewInferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InferenceLogic {
	return &InferenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (l *InferenceLogic) Inference(req *types.InferenceRequest) (resp *types.InferenceResponse, err error) {
	// 1. 构造 InferenceService 名称（与创建时保持一致：{modelName}-{modelVersion}）
	modelName := req.ModelName
	modelVersion := req.ModelVersion
	if modelVersion == "" {
		modelVersion = "latest" // 默认版本
	}
	isvcName := fmt.Sprintf("%s-%s", modelName, modelVersion)

	// 2. 获取 InferenceService CR 对象
	isvc := &aiv1.InferenceService{}
	err = l.svcCtx.CtrlClient.Get(l.ctx, k8stype.NamespacedName{
		Namespace: l.svcCtx.Config.K8s.Namespace,
		Name:      isvcName,
	}, isvc)
	if err != nil {
		logx.Errorf("InferenceService %s not found: %v", isvcName, err)
		return nil, fmt.Errorf("inference service '%s' not found or not ready", isvcName)
	}

	// 3. 检查服务是否就绪
	if !isvc.Status.Ready || isvc.Status.URL == "" {
		return nil, fmt.Errorf("inference service '%s' is not ready", isvcName)
	}

	// 4. 构造目标 URL（使用 status.url 或内部 Service 域名）
	targetURL := isvc.Status.URL
	if targetURL == "" {
		// 降级：直接构造 Service 域名
		targetURL = fmt.Sprintf("http://%s-stable.%s.svc.cluster.local:80", isvcName, l.svcCtx.Config.K8s.Namespace)
	}

	// 5. 构建转发请求
	proxyReq, err := l.buildProxyRequest(req, targetURL)
	if err != nil {
		return nil, err
	}

	// 6. 执行 HTTP 请求
	response, err := l.httpClient.Do(proxyReq)
	if err != nil {
		logx.Errorf("proxy request failed: %v", err)
		return nil, fmt.Errorf("inference request failed: %w", err)
	}
	defer response.Body.Close()

	// 7. 读取并解析响应
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inference service returned error: status=%d, body=%s", response.StatusCode, string(body))
	}

	// 8. 解析推理结果
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// 如果不是 JSON，包装成字符串结果
		result = map[string]interface{}{
			"output": string(body),
		}
	}
	// 9. 保存推理任务到数据库
	taskID := generateTaskID()

	l.svcCtx.InferenceTaskRepo.Create(l.ctx, &model.InferenceTask{
		TaskID:       taskID,
		Status:       model.StatusRunning,
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		InputData:    req.Input,
		OutputTopic:  result["output"].(string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Framework:    req.Framework,
	})

	return &types.InferenceResponse{
		TaskID: taskID,
		Result: result,
	}, nil
}

// buildProxyRequest 构建转发到推理服务的 HTTP 请求
func (l *InferenceLogic) buildProxyRequest(req *types.InferenceRequest, targetURL string) (*http.Request, error) {
	// 将请求体序列化为 JSON
	bodyBytes, err := json.Marshal(req.Input)
	if err != nil {
		return nil, fmt.Errorf("marshal input failed: %w", err)
	}

	// 解析目标 URL
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	// 构造请求路径（推理服务通常暴露 /v1/models/{model}:predict 或自定义路径）
	// 这里假设推理服务遵循 TensorFlow Serving 或 KServe 的 predict 协议
	predictPath := fmt.Sprintf("/v1/models/%s:predict", req.ModelName)
	target.Path = predictPath

	proxyReq, err := http.NewRequestWithContext(l.ctx, http.MethodPost, target.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	// 可透传部分原始请求头
	// for key, values := range r.Header { ... }

	return proxyReq, nil
}

// 生成简短任务 ID
func generateTaskID() string {
	return fmt.Sprintf("inf-%d", time.Now().UnixNano())
}
