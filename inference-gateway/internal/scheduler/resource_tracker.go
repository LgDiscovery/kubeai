package scheduler

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

// NodeResourceInfo 节点资源信息
type NodeResourceInfo struct {
	Name              string
	AllocatableCPU    int64 //豪核
	AllocatableMemory int64 //字节
	AllocatableGPU    int64 //卡数
	RequestedCPU      int64 //核
	RequestedMemory   int64 //字节
	RequestedGPU      int64 //卡数
	AvailableCPU      int64 //核数
	AvailableMemory   int64 //字节
	AvailableGPU      int64
	Labels            map[string]string
	LastUpdateTime    time.Time
}

// ResourceRequest 内部使用的资源需求（统一单位）
type ResourceRequest struct {
	CPUMilli     int64
	MemoryBytes  int64
	GPUCount     int64
	ModelName    string // 用于亲和性调度
	ModelVersion string
}

// ReservedResource 预留资源记录
type ReservedResource struct {
	TaskID      string
	CPUMilli    int64
	MemoryBytes int64
	GPUCount    int64
	ReservedAt  time.Time
	ExpiresAt   time.Time
}

// ResourceTracker 实时同步集群算力状态
type ResourceTracker struct {
	client            *kubernetes.Clientset
	namespace         string
	mu                sync.RWMutex
	nodes             map[string]*NodeResourceInfo
	reservedResources map[string]map[string]*ReservedResource // nodeName -> taskID -> ReservedResource
	stopCh            chan struct{}
	interval          time.Duration
	defaultStrategy   PlacementStrategy
}

func NewResourceTracker(client *kubernetes.Clientset, namespace string, interval time.Duration, defaultStrategy PlacementStrategy) *ResourceTracker {
	rt := &ResourceTracker{
		client:            client,
		namespace:         namespace,
		interval:          interval,
		stopCh:            make(chan struct{}),
		nodes:             make(map[string]*NodeResourceInfo),
		reservedResources: make(map[string]map[string]*ReservedResource),
		defaultStrategy:   defaultStrategy,
	}
	go rt.run()
	go rt.cleanupExpiredReservations() // 启动过期预留清理协程
	return rt
}

func (rt *ResourceTracker) run() {
	ticker := time.NewTicker(rt.interval)
	defer ticker.Stop()
	for {
		select {
		case <-rt.stopCh:
			return
		case <-ticker.C:
			if err := rt.sync(); err != nil {
				logx.Errorf("sync node resources failed: %v", err)
			}
		}
	}
}

func (rt *ResourceTracker) sync() error {
	ctx := context.Background()
	// 获取所有节点
	nodes, err := rt.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	// 获取所有 Pod（用于计算已请求资源）
	pods, err := rt.client.CoreV1().Pods(rt.namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase!=Succeeded,status.phase!=Failed",
	})
	if err != nil {
		return err
	}

	newNodes := make(map[string]*NodeResourceInfo)
	for _, node := range nodes.Items {
		info := &NodeResourceInfo{
			Name:              node.Name,
			AllocatableCPU:    node.Status.Allocatable.Cpu().MilliValue(),
			AllocatableMemory: node.Status.Allocatable.Memory().Value(),
			AllocatableGPU:    getGPUCount(node.Status.Allocatable),
			Labels:            node.Labels,
			LastUpdateTime:    time.Now(),
		}
		// 累加该节点上 Pod 的请求
		for _, pod := range pods.Items {
			if pod.Spec.NodeName != node.Name {
				continue
			}
			for _, container := range pod.Spec.Containers {
				info.RequestedCPU += container.Resources.Requests.Cpu().MilliValue()
				info.RequestedMemory += container.Resources.Requests.Memory().Value()
				info.RequestedGPU += getGPUCount(container.Resources.Requests)
			}
		}

		// 2. 累加预留资源
		rt.mu.RLock()
		if reservations, ok := rt.reservedResources[node.Name]; ok {
			for _, res := range reservations {
				info.RequestedCPU += res.CPUMilli
				info.RequestedMemory += res.MemoryBytes
				info.RequestedGPU += res.GPUCount
			}
		}
		rt.mu.RUnlock()
		// 3. 计算可用资源
		info.AvailableCPU = info.AllocatableCPU - info.RequestedCPU
		info.AvailableMemory = info.AllocatableMemory - info.RequestedMemory
		info.AvailableGPU = info.AllocatableGPU - info.RequestedGPU
		newNodes[node.Name] = info
	}

	rt.mu.Lock()
	rt.nodes = newNodes
	rt.mu.Unlock()
	logx.Infof("resource sync completed, nodes: %d", len(newNodes))
	return nil
}

// GetNodes 获取所有节点资源快照
func (rt *ResourceTracker) GetNodes() []*NodeResourceInfo {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	result := make([]*NodeResourceInfo, 0, len(rt.nodes))
	for _, node := range rt.nodes {
		result = append(result, node)
	}
	return result
}

// FindFitNode 根据资源需求寻找最合适的节点（使用调度策略）
func (rt *ResourceTracker) FindFitNode(req ResourceRequest, strategy PlacementStrategy) (string, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	nodes := make([]*NodeResourceInfo, 0, len(rt.nodes))
	for _, node := range rt.nodes {
		// 检查资源是否满足
		if node.AvailableCPU < req.CPUMilli || node.AvailableMemory < req.MemoryBytes || node.AvailableGPU < req.GPUCount {
			continue
		}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return "", ErrInsufficientResource
	}
	selected := strategy.SelectNode(nodes, req)
	return selected.Name, nil
}

func getGPUCount(rl corev1.ResourceList) int64 {
	if val, ok := rl["nvidia.com/gpu"]; ok {
		return val.Value()
	}
	return 0
}

// FindFitNodeAndReserve 筛选节点并预留资源，【核心实现】
func (rt *ResourceTracker) FindFitNodeAndReserve(req ResourceRequest, strategy PlacementStrategy, taskID string) (string, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	selectedStrategy := strategy
	if selectedStrategy == nil {
		selectedStrategy = rt.defaultStrategy
	}

	fitNodes := rt.getFitNodesLocked(req)
	if len(fitNodes) == 0 {
		return "", ErrInsufficientResource
	}

	selected := selectedStrategy.SelectNode(fitNodes, req)
	nodeName := selected.Name

	// 执行资源预留
	reservation := &ReservedResource{
		TaskID:      taskID,
		CPUMilli:    req.CPUMilli,
		MemoryBytes: req.MemoryBytes,
		GPUCount:    req.GPUCount,
		ReservedAt:  time.Now(),
		ExpiresAt:   time.Now().Add(5 * time.Minute), // 预留5分钟过期
	}

	if _, ok := rt.reservedResources[nodeName]; !ok {
		rt.reservedResources[nodeName] = make(map[string]*ReservedResource)
	}
	rt.reservedResources[nodeName][taskID] = reservation

	logx.Infof("node reserve success, node: %s, task: %s, cpu: %dm, mem: %d, gpu: %d",
		nodeName, taskID, req.CPUMilli, req.MemoryBytes, req.GPUCount)

	return nodeName, nil
}

// ReleaseNodeReserve 释放节点预留资源
func (rt *ResourceTracker) ReleaseNodeReserve(nodeName string, taskID string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if nodeReservations, ok := rt.reservedResources[nodeName]; ok {
		if res, ok := nodeReservations[taskID]; ok {
			delete(nodeReservations, taskID)
			logx.Infof("node reserve released, node: %s, task: %s, cpu: %dm, mem: %d, gpu: %d",
				nodeName, taskID, res.CPUMilli, res.MemoryBytes, res.GPUCount)
			// 如果该节点没有预留了，清理map
			if len(nodeReservations) == 0 {
				delete(rt.reservedResources, nodeName)
			}
			return nil
		}
	}

	logx.Errorf("node reserve not found, node: %s, task: %s", nodeName, taskID)
	return nil
}

// Stop 停止资源追踪
func (rt *ResourceTracker) Stop() {
	close(rt.stopCh)
}

// ==================== 内部辅助方法 ====================

func (rt *ResourceTracker) getFitNodesLocked(req ResourceRequest) []*NodeResourceInfo {
	nodes := make([]*NodeResourceInfo, 0, len(rt.nodes))
	for _, node := range rt.nodes {
		// 检查资源是否满足（包含预留后的可用资源）
		if node.AvailableCPU < req.CPUMilli ||
			node.AvailableMemory < req.MemoryBytes ||
			node.AvailableGPU < req.GPUCount {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func (rt *ResourceTracker) cleanupExpiredReservations() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rt.stopCh:
			return
		case <-ticker.C:
			rt.mu.Lock()
			now := time.Now()
			for nodeName, reservations := range rt.reservedResources {
				for taskID, res := range reservations {
					if now.After(res.ExpiresAt) {
						delete(reservations, taskID)
						logx.Infof("cleanup expired reservation, node: %s, task: %s", nodeName, taskID)
					}
				}
				if len(reservations) == 0 {
					delete(rt.reservedResources, nodeName)
				}
			}
			rt.mu.Unlock()
		}
	}
}

var ErrInsufficientResource = fmt.Errorf("insufficient resources in cluster")
