package discovery

import (
	"errors"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc/resolver"
	"sync"
	"sync/atomic"
)

type ServiceDiscovery struct {
	mu                      sync.RWMutex
	etcdHosts               []string
	modelManagerSub         *discov.Subscriber
	jobSchedulerSub         *discov.Subscriber
	inferenceGatewaySub     *discov.Subscriber
	observabilityGatewaySub *discov.Subscriber

	// 每个服务独立的轮询计数器（高并发安全）
	rrModelManager         atomic.Uint32
	rrJobScheduler         atomic.Uint32
	rrInferenceGateway     atomic.Uint32
	rrObservabilityGateway atomic.Uint32
}

func NewServiceDiscovery(etcdHosts []string, modelManagerKey, jobSchedulerKey, inferenceGatewayKey, observabilityGatewayKey string) *ServiceDiscovery {

	sd := &ServiceDiscovery{
		etcdHosts: etcdHosts,
	}
	var err error
	// 创建 subscriber
	sd.modelManagerSub, err = discov.NewSubscriber(etcdHosts, modelManagerKey)
	if err != nil {
		panic(err)
	}
	sd.jobSchedulerSub, err = discov.NewSubscriber(etcdHosts, jobSchedulerKey)
	if err != nil {
		panic(err)
	}
	sd.inferenceGatewaySub, err = discov.NewSubscriber(etcdHosts, inferenceGatewayKey)
	if err != nil {
		panic(err)
	}
	sd.observabilityGatewaySub, err = discov.NewSubscriber(etcdHosts, observabilityGatewayKey)
	if err != nil {
		panic(err)
	}

	// 注册 resolver，让 gRPC 客户端可以使用 etcd 服务发现
	resolver.Register()

	return sd
}

// GetModelManagerEndpoint 获取模型管理服务的当前可用端点（支持轮询负载均衡）
func (s *ServiceDiscovery) GetModelManagerEndpoint() (string, error) {
	values := s.modelManagerSub.Values()
	if len(values) == 0 {
		return "", errors.New("no model manager endpoint")
	}
	// 原子轮询
	next := s.rrModelManager.Add(1) - 1
	idx := next % uint32(len(values))
	return values[idx], nil
}

// GetTaskSchedulerEndpoint 获取任务调度端点
func (s *ServiceDiscovery) GetTaskSchedulerEndpoint() (string, error) {
	values := s.jobSchedulerSub.Values()
	if len(values) == 0 {
		return "", errors.New("no task scheduler endpoint")
	}
	next := s.rrJobScheduler.Add(1) - 1
	idx := next % uint32(len(values))
	return values[idx], nil
}

// GetInferenceGatewayEndpoint 获取推理网关端点
func (s *ServiceDiscovery) GetInferenceGatewayEndpoint() (string, error) {
	values := s.inferenceGatewaySub.Values()
	if len(values) == 0 {
		return "", errors.New("no inference gateway endpoint")
	}
	next := s.rrInferenceGateway.Add(1) - 1
	idx := next % uint32(len(values))
	return values[idx], nil
}

// GetObservabilityGatewayEndpoint 获取可观测网关端点
func (s *ServiceDiscovery) GetObservabilityGatewayEndpoint() (string, error) {
	values := s.observabilityGatewaySub.Values()
	if len(values) == 0 {
		return "", errors.New("no observability gateway endpoint")
	}
	next := s.rrObservabilityGateway.Add(1) - 1
	idx := next % uint32(len(values))
	return values[idx], nil
}

// Watch 监听服务端点变化（用于日志或内部状态）
func (s *ServiceDiscovery) Watch() {
	s.modelManagerSub.AddListener(func() {
		vals := s.modelManagerSub.Values()
		logx.Infof("model-manager endpoints changed: %v", vals)
	})
	s.jobSchedulerSub.AddListener(func() {
		vals := s.jobSchedulerSub.Values()
		logx.Infof("job-scheduler endpoints changed: %v", vals)
	})
	s.inferenceGatewaySub.AddListener(func() {
		vals := s.inferenceGatewaySub.Values()
		logx.Infof("inference-gateway endpoints changed: %v", vals)
	})
	s.observabilityGatewaySub.AddListener(func() {
		vals := s.observabilityGatewaySub.Values()
		logx.Infof("observability-gateway endpoints changed: %v", vals)
	})
}
