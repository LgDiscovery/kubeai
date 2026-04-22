package scheduler

import (
	"sort"
	"strings"
)

// PlacementStrategy 节点选择策略接口
type PlacementStrategy interface {
	SelectNode(nodes []*NodeResourceInfo, req ResourceRequest) *NodeResourceInfo
}

// BinpackStrategy 装箱策略：选择资源最满的节点（提高利用率）
type BinpackStrategy struct {
	enableGPUPacking bool
	gpuWeight        float64
}

func NewBinpackStrategy(enableGPUPacking bool, gpuWeight float64) *BinpackStrategy {
	return &BinpackStrategy{enableGPUPacking: enableGPUPacking, gpuWeight: gpuWeight}
}

func (s *BinpackStrategy) SelectNode(nodes []*NodeResourceInfo, req ResourceRequest) *NodeResourceInfo {
	// 按资源使用率降序排列
	sort.Slice(nodes, func(i, j int) bool {
		scoreI := s.calculateScore(nodes[i])
		scoreJ := s.calculateScore(nodes[j])
		return scoreI > scoreJ
	})
	return nodes[0]
}

func (s *BinpackStrategy) calculateScore(node *NodeResourceInfo) float64 {
	cpuRatio := float64(node.RequestedCPU) / float64(node.AllocatableCPU)
	memRatio := float64(node.RequestedMemory) / float64(node.AllocatableMemory)
	score := (cpuRatio + memRatio) / 2
	if s.enableGPUPacking && node.AllocatableGPU > 0 {
		gpuRatio := float64(node.RequestedGPU) / float64(node.AllocatableGPU)
		score = score*(1-s.gpuWeight) + gpuRatio*s.gpuWeight
	}
	return score
}

// SpreadStrategy 分散策略：选择资源最空闲的节点（提高可用性）
type SpreadStrategy struct{}

func NewSpreadStrategy() *SpreadStrategy {
	return &SpreadStrategy{}
}

func (s *SpreadStrategy) SelectNode(nodes []*NodeResourceInfo, req ResourceRequest) *NodeResourceInfo {
	sort.Slice(nodes, func(i, j int) bool {
		cpuFreeI := nodes[i].AvailableCPU
		cpuFreeJ := nodes[j].AvailableCPU
		return cpuFreeI > cpuFreeJ
	})
	return nodes[0]
}

// GPUAffinityStrategy GPU 亲和性调度：优先选择已有指定模型缓存的节点
type GPUAffinityStrategy struct {
	// 模型缓存标签键前缀，例如 "kubeai.io/model-cache."
	// 实际标签格式：prefix + modelName + ":" + modelVersion 或者只是 modelName
	cacheLabelPrefix string
	// 当没有匹配缓存节点时，使用的回退策略
	fallback PlacementStrategy
}

// NewGPUAffinityStrategy 创建 GPU 亲和策略
// cacheLabelPrefix: 模型缓存标签前缀，例如 "kubeai.io/model-cache."
// fallback: 无缓存节点时使用的调度策略（如 Binpack）
func NewGPUAffinityStrategy(cacheLabelPrefix string, fallback PlacementStrategy) *GPUAffinityStrategy {
	if fallback == nil {
		fallback = NewBinpackStrategy(true, 0.7)
	}
	return &GPUAffinityStrategy{
		cacheLabelPrefix: cacheLabelPrefix,
		fallback:         fallback,
	}
}

// SelectNode 实现节点选择
func (s *GPUAffinityStrategy) SelectNode(nodes []*NodeResourceInfo, req ResourceRequest) *NodeResourceInfo {
	// 分离出有模型缓存标签的节点和无缓存的节点
	cachedNodes := make([]*NodeResourceInfo, 0)
	otherNodes := make([]*NodeResourceInfo, 0)

	for _, node := range nodes {
		if s.hasModelCache(node, req.ModelName, req.ModelVersion) {
			cachedNodes = append(cachedNodes, node)
		} else {
			otherNodes = append(otherNodes, node)
		}
	}

	// 如果有缓存节点，在缓存节点中按资源使用率排序（装箱）
	if len(cachedNodes) > 0 {
		// 使用 Binpack 策略在缓存节点中选择
		binpack := NewBinpackStrategy(true, 0.7)
		return binpack.SelectNode(cachedNodes, req)
	}

	// 无缓存节点，使用回退策略
	if len(otherNodes) > 0 {
		return s.fallback.SelectNode(otherNodes, req)
	}

	// 理论上不会走到这里，因为 nodes 非空
	return nil
}

// hasModelCache 判断节点是否有指定模型的缓存标签
// 支持两种格式：
// 1. 前缀 + modelName + "/" + version，例如 "kubeai.io/model-cache.bert-classifier/v1"
// 2. 前缀 + modelName，例如 "kubeai.io/model-cache.bert-classifier"
func (s *GPUAffinityStrategy) hasModelCache(node *NodeResourceInfo, modelName, version string) bool {
	if node.Labels == nil {
		return false
	}
	// 精确匹配: prefix + modelName + "/" + version
	exactKey := s.cacheLabelPrefix + modelName + "/" + version
	if _, ok := node.Labels[exactKey]; ok {
		return true
	}
	// 模糊匹配: prefix + modelName (任意版本)
	modelKey := s.cacheLabelPrefix + modelName
	for k := range node.Labels {
		if strings.HasPrefix(k, modelKey) {
			return true
		}
	}
	return false
}
