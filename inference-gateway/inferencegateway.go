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

	ctx := svc.NewServiceContext(c)
	// 初始化队列消费者组
	if err := ctx.InferenceQueue.Init(context.Background()); err != nil {
		logx.Must(err)
	}
	if err := ctx.TrainingQueue.Init(context.Background()); err != nil {
		logx.Must(err)
	}

	// 启动消费者
	go startInferenceConsumer(ctx)
	go startTrainingConsumer(ctx)

	// 启动控制器管理器
	go func() {
		if err := ctx.Mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			logx.Must(err)
		}
	}()

	// 注册控制器
	if err := (&controllers.InferenceServiceReconciler{
		Client:           ctx.Mgr.GetClient(),
		Scheme:           ctx.Mgr.GetScheme(),
		ModelManagerAddr: c.ModelManager.URL,
	}).SetupWithManager(ctx.Mgr); err != nil {
		logx.Must(err)
	}
	if err := (&controllers.TrainingJobReconciler{
		Client: ctx.Mgr.GetClient(),
		Scheme: ctx.Mgr.GetScheme(),
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
		server.Stop()
	}()

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

func startInferenceConsumer(ctx *svc.ServiceContext) {
	consumerLogic := logic.NewConsumerLogic(context.Background(), ctx)
	ctx.InferenceQueue.Consume(context.Background(), "inference-consumer",
		func(taskID string, data []byte) error {
			return consumerLogic.ProcessInferenceTask(taskID, data)
		})
}

func startTrainingConsumer(ctx *svc.ServiceContext) {
	consumerLogic := logic.NewConsumerLogic(context.Background(), ctx)
	ctx.TrainingQueue.Consume(context.Background(), "training-consumer",
		func(taskID string, data []byte) error {
			return consumerLogic.ProcessTrainingTask(taskID, data)
		})
}
