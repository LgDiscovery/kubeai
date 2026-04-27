package help

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"kubeai-inference-gateway/internal/model"
	"kubeai-inference-gateway/internal/scheduler"
	"math/rand"
	"path/filepath"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func ConvertToResourceRequest(res model.ResourceRequest) scheduler.ResourceRequest {
	cpu, _ := resource.ParseQuantity(res.CPU)
	memory, _ := resource.ParseQuantity(res.Memory)
	gpu := int64(0)
	if res.GPU != "" {
		gpuQ, _ := resource.ParseQuantity(res.GPU)
		gpu = gpuQ.Value()
	}

	return scheduler.ResourceRequest{
		CPUMilli:    cpu.MilliValue(),
		MemoryBytes: memory.Value(),
		GPUCount:    gpu,
	}
}

// RandomString 随机字符串
func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// LoadKubeConfig 加载 kubeconfig 集群内/集群外自动适配
func LoadKubeConfig() (*rest.Config, error) {
	// 集群内
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// 集群外 ~/.kube/config
	home := homedir.HomeDir()
	kubeConfigPath := filepath.Join(home, ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
}
