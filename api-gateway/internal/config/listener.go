package config

import (
	configcenter "github.com/zeromicro/go-zero/core/configcenter"
	"github.com/zeromicro/go-zero/core/configcenter/subscriber"
	"github.com/zeromicro/go-zero/core/logx"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// StartConfigListener 启动配置监听，当 etcd 中配置变化时更新 HotConfig
func StartConfigListener(etcdConf EtcdConf, key string, callback func(HotConfig)) error {
	// 创建 etcd 客户端
	_, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdConf.Hosts,
		Username:    etcdConf.User,
		Password:    etcdConf.Pass,
		DialTimeout: etcdConf.DialTimeout,
	})
	if err != nil {
		return err
	}

	// 创建 etcd subscriber
	sub := subscriber.MustNewEtcdSubscriber(subscriber.EtcdConf{
		Hosts: etcdConf.Hosts,
		Key:   key,
	})

	// 创建配置中心
	cc := configcenter.MustNewConfigCenter[HotConfig](configcenter.Config{
		Type: "json",
	}, sub)

	// 添加监听
	cc.AddListener(func() {
		if hotCfg, err := cc.GetConfig(); err == nil {
			logx.Infof("hot config changed: %+v", hotCfg)
			callback(hotCfg)
		}
	})

	return nil
}
