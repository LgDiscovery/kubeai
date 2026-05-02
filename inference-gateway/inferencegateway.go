// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"syscall"
	"time"

	"kubeai-inference-gateway/internal/config"
	"kubeai-inference-gateway/internal/handler"
	"kubeai-inference-gateway/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	controllers "kubeai-inference-gateway/internal/controllers"
	logic "kubeai-inference-gateway/internal/logic/inference"
)

var configFile = flag.String("f", "etc/inference-gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 手动注册服务到 ETCD
	pub := discov.NewPublisher(
		c.Etcd.Hosts,
		c.Etcd.Key,
		fmt.Sprintf("%s:%d", c.Host, c.Port), // 本服务监听地址
	)
	defer pub.Stop()
	logx.Infof("✅ 服务已注册到 etcd: %s", c.Etcd.Key)
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := svc.NewServiceContext(c)
	// 初始化队列消费者组
	if err := ctx.InferenceQueue.Init(rootCtx); err != nil {
		logx.Must(err)
	}
	if err := ctx.TrainingQueue.Init(rootCtx); err != nil {
		logx.Must(err)
	}
	if err := ctx.DeadLetterQueue.Init(rootCtx); err != nil {
		logx.Must(err)
	}

	// 启动消费者
	Start(rootCtx, ctx)

	// 启动控制器管理器
	go func() {
		if err := ctx.Mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			logx.Must(err)
		}
	}()

	// 注册控制器
	if err := (&controllers.InferenceServiceReconciler{
		Client:            ctx.Mgr.GetClient(),
		Scheme:            ctx.Mgr.GetScheme(),
		ModelManagerAddr:  c.ModelManager.URL,
		JobScheduleAddr:   c.JobSchedule.URL,
		ModelClient:       ctx.ModelMgrClient,
		JobScheduleClient: ctx.JobScheduleClient,
	}).SetupWithManager(ctx.Mgr); err != nil {
		logx.Must(err)
	}
	if err := (&controllers.TrainingJobReconciler{
		Client:            ctx.Mgr.GetClient(),
		Scheme:            ctx.Mgr.GetScheme(),
		ModelManagerAddr:  c.ModelManager.URL,
		JobScheduleAddr:   c.JobSchedule.URL,
		ModelClient:       ctx.ModelMgrClient,
		JobScheduleClient: ctx.JobScheduleClient,
	}).SetupWithManager(ctx.Mgr); err != nil {
		logx.Must(err)
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()
	handler.RegisterHandlers(server, ctx)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		logx.Info("Shutting down inference-gateway...")
		// 取消所有 goroutine
		cancel()
		server.Stop()
	}()

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

// Start 启动消费者
func Start(rootCtx context.Context, ctx *svc.ServiceContext) {
	go startInferenceConsumer(rootCtx, ctx)
	go startTrainingConsumer(rootCtx, ctx)
	go startDeadLetterConsumer(rootCtx, ctx)
	go startInferencePendingClaim(rootCtx, ctx)
	go startTrainingPendingClaim(rootCtx, ctx)
	<-rootCtx.Done()
	logx.Infof("inference task consumer stopped")
	logx.Infof("training task consumer stopped")
	logx.Infof("dead letter task consumer stopped")
	logx.Infof("inference pending claim task consumer stopped")
	logx.Infof("training pending claim task consumer stopped")
}

func startInferenceConsumer(rootCtx context.Context, ctx *svc.ServiceContext) {
	consumerLogic := logic.NewConsumerLogic(rootCtx, ctx, ctx.Config.Redis.Streams.Inference)
	ctx.InferenceQueue.Consume(rootCtx, "inference-consumer",
		func(taskID string, data []byte) error {
			return consumerLogic.ProcessInferenceTask(taskID, data)
		})
}

func startTrainingConsumer(rootCtx context.Context, ctx *svc.ServiceContext) {
	consumerLogic := logic.NewConsumerLogic(rootCtx, ctx, ctx.Config.Redis.Streams.Training)
	ctx.TrainingQueue.Consume(rootCtx, "training-consumer",
		func(taskID string, data []byte) error {
			return consumerLogic.ProcessTrainingTask(taskID, data)
		})
}

func startDeadLetterConsumer(rootCtx context.Context, ctx *svc.ServiceContext) {
	consumerLogic := logic.NewConsumerLogic(rootCtx, ctx, ctx.Config.Redis.Streams.DeadLetter)
	ctx.DeadLetterQueue.Pop(rootCtx, "dead-letter-consumer", ctx.Config.Redis.MaxRetries,
		func(taskID string, data []byte, taskType string) error {
			if taskType == "training" {
				return consumerLogic.ProcessTrainingTask(taskID, data)
			} else {
				return consumerLogic.ProcessInferenceTask(taskID, data)
			}
		})
}

// startInferencePendingClaim 推理队列 超时消息认领
func startInferencePendingClaim(rootCtx context.Context, ctx *svc.ServiceContext) {
	// 超时时间：30秒未处理的消息，自动认领重试
	const minIdle = 30 * time.Second
	consumerLogic := logic.NewConsumerLogic(rootCtx, ctx, ctx.Config.Redis.Streams.Inference)

	// 调用ClaimPending，处理失败则推入死信队列
	ctx.InferenceQueue.ClaimPending(rootCtx, "inference-pending-claimer", minIdle,
		func(taskID string, data []byte) error {
			// 执行业务逻辑
			err := consumerLogic.ProcessInferenceTask(taskID, data)
			if err != nil {
				// 🔥 重试失败：推入死信队列
				logx.Errorf("推理任务[%s]重试失败，加入死信队列", taskID)
				_ = ctx.DeadLetterQueue.Push(rootCtx, taskID, data, "inference_pending_timeout", "inference")
			}
			return err
		})
}

// startTrainingPendingClaim 训练队列 超时消息认领
func startTrainingPendingClaim(rootCtx context.Context, ctx *svc.ServiceContext) {
	// 超时时间：30秒未处理的消息，自动认领重试
	const minIdle = 30 * time.Second
	consumerLogic := logic.NewConsumerLogic(rootCtx, ctx, ctx.Config.Redis.Streams.Training)

	// 调用ClaimPending，处理失败则推入死信队列
	ctx.TrainingQueue.ClaimPending(rootCtx, "training-pending-claimer", minIdle,
		func(taskID string, data []byte) error {
			// 执行业务逻辑
			err := consumerLogic.ProcessTrainingTask(taskID, data)
			if err != nil {
				// 🔥 重试失败：推入死信队列
				logx.Errorf("训练任务[%s]重试失败，加入死信队列", taskID)
				_ = ctx.DeadLetterQueue.Push(rootCtx, taskID, data, "training_pending_timeout", "training")
			}
			return err
		})
}
