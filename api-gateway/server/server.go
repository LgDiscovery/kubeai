package server

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"kubeai-api-gateway/internal/config"
	"kubeai-api-gateway/internal/handler"
	"kubeai-api-gateway/internal/svc"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func Start() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)

	// 1.注册自动生成的路由
	handler.RegisterHandlers(server, ctx)

	// 2. 手动注册通配业务代理路由，覆盖所有方法和多段路径
	handler.RegisterCommHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
