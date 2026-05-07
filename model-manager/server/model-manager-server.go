package server

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-model-manager/internal/config"
	"kubeai-model-manager/internal/handler"
	"kubeai-model-manager/internal/svc"
	"net/http"
)

var configFile = flag.String("f", "etc/model-manager-service.yaml", "the config file")

func Start() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 手动注册服务到 ETCD
	pub := discov.NewPublisher(
		c.Etcd.Hosts,
		c.Etcd.Key,
		fmt.Sprintf("%s:%d", c.Host, c.Port), // 本服务监听地址
	)
	defer pub.Stop()
	logx.Infof("✅ 服务已注册到 etcd: %s", c.Etcd.Key)

	// 自定义错误处理
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		return http.StatusInternalServerError, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
