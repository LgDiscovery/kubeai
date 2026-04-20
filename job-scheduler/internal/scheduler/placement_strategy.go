package scheduler

import (
	"sort"
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

// GPUAffinityStrategy GPU 亲和性调度：优先选择已有相同模型缓存的节点（需扩展缓存追踪）
// 此处提供框架，生产环境可集成节点缓存标签
/*type GPUAffinityStrategy struct{}

func NewGPUAffinityStrategy() *GPUAffinityStrategy {
	return &GPUAffinityStrategy{}
}

func (s *GPUAffinityStrategy) SelectNode(nodes []*NodeResourceInfo, req ResourceRequest) *NodeResourceInfo {
	// 按节点缓存标签排序
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CacheTags[req.ModelName] > nodes[j].CacheTags[req.ModelName]
	})
	return nodes[0]
}*/
