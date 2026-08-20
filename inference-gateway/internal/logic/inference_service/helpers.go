package inference_service

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// buildResourceRequirements 构建 K8s 资源需求
func buildResourceRequirements(cpuStr, memoryStr, gpuStr string) (corev1.ResourceRequirements, error) {
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	// CPU
	if cpuStr != "" {
		cpuQty, err := parseQuantity(cpuStr)
		if err != nil {
			return resources, fmt.Errorf("CPU 配置无效: %w", err)
		}
		resources.Requests[corev1.ResourceCPU] = cpuQty
		resources.Limits[corev1.ResourceCPU] = cpuQty
	}

	// 内存
	if memoryStr != "" {
		memQty, err := parseQuantity(memoryStr)
		if err != nil {
			return resources, fmt.Errorf("内存配置无效: %w", err)
		}
		resources.Requests[corev1.ResourceMemory] = memQty
		resources.Limits[corev1.ResourceMemory] = memQty
	}

	// GPU
	if gpuStr != "" {
		gpuQty, err := parseQuantity(gpuStr)
		if err != nil {
			return resources, fmt.Errorf("GPU 配置无效: %w", err)
		}
		resources.Requests["nvidia.com/gpu"] = gpuQty
		resources.Limits["nvidia.com/gpu"] = gpuQty
	}

	return resources, nil
}

// parseQuantity 解析资源数量，支持纯数字（自动判断单位）和带单位的字符串
func parseQuantity(str string) (resource.Quantity, error) {
	str = strings.TrimSpace(str)
	if str == "" {
		return resource.Quantity{}, fmt.Errorf("空值")
	}

	// 尝试直接解析（支持带单位如 "2", "4Gi", "500m"）
	qty, err := resource.ParseQuantity(str)
	if err == nil {
		return qty, nil
	}

	// 如果是纯数字，CPU 默认单位为核，内存默认单位为 Mi
	if _, err := strconv.ParseFloat(str, 64); err == nil {
		// 纯数字，直接作为资源数量（CPU 为核数，内存需要调用方指定单位）
		return resource.MustParse(str), nil
	}

	return resource.Quantity{}, fmt.Errorf("无法解析资源数量: %s", str)
}
