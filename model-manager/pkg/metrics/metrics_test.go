package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestRequestTotal_MetricsType(t *testing.T) {
	// 验证 RequestTotal 是 CounterVec 类型
	// CounterVec 应该有 Inc() 方法
	RequestTotal.WithLabelValues("GET", "/test", "200", "model-manager").Inc()

	// 收集指标并验证
	metricCh := make(chan prometheus.Metric, 1)
	RequestTotal.WithLabelValues("GET", "/test", "200", "model-manager").Write(metricCh)
	metric := <-metricCh

	dtoMetric := &dto.Metric{}
	err := metric.Write(dtoMetric)
	assert.NoError(t, err)

	// Counter 类型应该有 Counter 字段
	assert.NotNil(t, dtoMetric.Counter)
	assert.NotNil(t, dtoMetric.Counter.Value)
	assert.Greater(t, *dtoMetric.Counter.Value, float64(0))
}

func TestRequestDuration_HistogramType(t *testing.T) {
	// 验证 RequestDuration 是 HistogramVec 类型
	RequestDuration.WithLabelValues("GET", "/test", "model-manager").Observe(0.1)

	// 收集指标并验证
	metricCh := make(chan prometheus.Metric, 1)
	RequestDuration.WithLabelValues("GET", "/test", "model-manager").Write(metricCh)
	metric := <-metricCh

	dtoMetric := &dto.Metric{}
	err := metric.Write(dtoMetric)
	assert.NoError(t, err)

	// Histogram 类型应该有 Histogram 字段
	assert.NotNil(t, dtoMetric.Histogram)
	assert.NotNil(t, dtoMetric.Histogram.SampleCount)
	assert.Greater(t, *dtoMetric.Histogram.SampleCount, uint64(0))
}

func TestModelTotal_GaugeType(t *testing.T) {
	// 验证 ModelTotal 是 Gauge 类型
	ModelTotal.Set(10)

	// 收集指标并验证
	metricCh := make(chan prometheus.Metric, 1)
	ModelTotal.Write(metricCh)
	metric := <-metricCh

	dtoMetric := &dto.Metric{}
	err := metric.Write(dtoMetric)
	assert.NoError(t, err)

	// Gauge 类型应该有 Gauge 字段
	assert.NotNil(t, dtoMetric.Gauge)
	assert.NotNil(t, dtoMetric.Gauge.Value)
	assert.Equal(t, float64(10), *dtoMetric.Gauge.Value)
}

func TestModelVersionTotal_Labels(t *testing.T) {
	// 验证 ModelVersionTotal 有 status 标签
	ModelVersionTotal.WithLabelValues("active").Inc()
	ModelVersionTotal.WithLabelValues("stopped").Inc()

	// 验证可以正确获取不同标签的指标
	metricCh := make(chan prometheus.Metric, 1)
	ModelVersionTotal.WithLabelValues("active").Write(metricCh)
	metric := <-metricCh

	dtoMetric := &dto.Metric{}
	err := metric.Write(dtoMetric)
	assert.NoError(t, err)

	// 验证标签
	assert.Len(t, dtoMetric.Label, 1)
	assert.Equal(t, "status", *dtoMetric.Label[0].Name)
	assert.Equal(t, "active", *dtoMetric.Label[0].Value)
}

func TestModelUploadTotal_CounterType(t *testing.T) {
	// 验证 ModelUploadTotal 是 CounterVec 类型
	ModelUploadTotal.WithLabelValues("test-model", "v1.0.0", "success").Inc()

	metricCh := make(chan prometheus.Metric, 1)
	ModelUploadTotal.WithLabelValues("test-model", "v1.0.0", "success").Write(metricCh)
	metric := <-metricCh

	dtoMetric := &dto.Metric{}
	err := metric.Write(dtoMetric)
	assert.NoError(t, err)

	assert.NotNil(t, dtoMetric.Counter)
	assert.Len(t, dtoMetric.Label, 3)
}

func TestHandler_ReturnsValidHandler(t *testing.T) {
	// 验证 Handler() 返回有效的 http.Handler
	handler := Handler()
	assert.NotNil(t, handler)
}
