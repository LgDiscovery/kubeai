package help

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"kubeai-inference-gateway/internal/model"
	"kubeai-inference-gateway/internal/scheduler"
)

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
