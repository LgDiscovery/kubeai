package inference_service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildResourceRequirements_ValidCPU(t *testing.T) {
	// 测试有效的 CPU 配置
	resources, err := buildResourceRequirements("2", "4Gi", "1")
	assert.NoError(t, err)
	assert.NotNil(t, resources)

	// 验证 CPU 请求
	cpuQty := resources.Requests[corev1.ResourceCPU]
	assert.Equal(t, "2", cpuQty.String())

	// 验证内存请求
	memQty := resources.Requests[corev1.ResourceMemory]
	assert.Equal(t, "4Gi", memQty.String())

	// 验证 GPU 请求
	gpuQty := resources.Requests["nvidia.com/gpu"]
	assert.Equal(t, "1", gpuQty.String())
}

func TestBuildResourceRequirements_EmptyConfig(t *testing.T) {
	// 测试空配置
	resources, err := buildResourceRequirements("", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, resources)

	// 空配置应该没有资源请求
	assert.Empty(t, resources.Requests)
	assert.Empty(t, resources.Limits)
}

func TestBuildResourceRequirements_OnlyCPU(t *testing.T) {
	// 测试只配置 CPU
	resources, err := buildResourceRequirements("4", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, resources)

	cpuQty := resources.Requests[corev1.ResourceCPU]
	assert.Equal(t, "4", cpuQty.String())

	// 内存和 GPU 应该不存在
	_, hasMem := resources.Requests[corev1.ResourceMemory]
	assert.False(t, hasMem)

	_, hasGPU := resources.Requests["nvidia.com/gpu"]
	assert.False(t, hasGPU)
}

func TestBuildResourceRequirements_InvalidCPU(t *testing.T) {
	// 测试无效的 CPU 配置
	_, err := buildResourceRequirements("invalid-cpu", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CPU 配置无效")
}

func TestBuildResourceRequirements_InvalidMemory(t *testing.T) {
	// 测试无效的内存配置
	_, err := buildResourceRequirements("", "invalid-memory", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "内存配置无效")
}

func TestParseQuantity_ValidValues(t *testing.T) {
	// 测试有效的资源数量
	tests := []struct {
		input    string
		expected string
	}{
		{"2", "2"},
		{"500m", "500m"},
		{"4Gi", "4Gi"},
		{"512Mi", "512Mi"},
		{"1", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			qty, err := parseQuantity(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, qty.String())
		})
	}
}

func TestParseQuantity_EmptyString(t *testing.T) {
	// 测试空字符串
	_, err := parseQuantity("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "空值")
}

func TestParseQuantity_Whitespace(t *testing.T) {
	// 测试带空格的字符串
	qty, err := parseQuantity("  2  ")
	assert.NoError(t, err)
	assert.Equal(t, "2", qty.String())
}

func TestFirstNonEmpty(t *testing.T) {
	// 测试 firstNonEmpty 函数
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"all empty", []string{"", "", ""}, ""},
		{"first non-empty", []string{"", "second", "third"}, "second"},
		{"all filled", []string{"first", "second", "third"}, "first"},
		{"single value", []string{"only"}, "only"},
		{"empty slice", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmpty(tt.values...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResourceQuantity_Comparison(t *testing.T) {
	// 测试资源数量比较
	qty1 := resource.MustParse("2")
	qty2 := resource.MustParse("4")

	assert.True(t, qty1.Cmp(qty2) < 0)
	assert.True(t, qty2.Cmp(qty1) > 0)
}

func TestResourceRequirements_LimitsMatchRequests(t *testing.T) {
	// 测试 Limits 和 Requests 是否匹配
	resources, err := buildResourceRequirements("2", "4Gi", "1")
	assert.NoError(t, err)

	// CPU Limits 应该等于 Requests
	assert.Equal(t, resources.Requests.Cpu().String(), resources.Limits.Cpu().String())
	// 内存 Limits 应该等于 Requests
	assert.Equal(t, resources.Requests.Memory().String(), resources.Limits.Memory().String())
}
