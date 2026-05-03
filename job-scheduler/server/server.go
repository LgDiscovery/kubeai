package server

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"kubeai-job-scheduler/internal/config"
	"kubeai-job-scheduler/internal/handler"
	"kubeai-job-scheduler/internal/svc"
)

var configFile = flag.String("f", "etc/job-scheduler.yaml", "the config file")

func Start() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	// 手动注册服务到 ETCD
	pub := discov.NewPublisher(
		c.Etcd.Hosts,
		c.Etcd.Key,
		fmt.Sprintf("%s:%d", c.Host, c.Port), // 本服务监听地址
	)
	defer pub.Stop()
	logx.Infof("✅ 服务已注册到 etcd: %s", c.Etcd.Key)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
