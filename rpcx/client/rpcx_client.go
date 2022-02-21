package client

import (
	"context"
	"time"

	etcd_client "github.com/rpcxio/rpcx-etcd/client"

	"github.com/smallnest/rpcx/client"
)

var (
	basePath string = "/zhangbin/rpcx"
)

type ClientManager struct {
	callTimeout time.Duration // 请求超时时间
	xc          client.XClient
}

func New(serviceName string, etcdAddr []string, callTimeout time.Duration, etcdBasePath string) *ClientManager {

	// 客户端和服务器的path要一致
	if etcdBasePath != "" {
		basePath = etcdBasePath
	}

	c := &ClientManager{
		callTimeout: callTimeout,
	}
	//NewEtcdV3Discovery指定basePath和etcd集群地址,方法路径(服务名称)
	d, _ := etcd_client.NewEtcdV3Discovery(basePath, serviceName, etcdAddr, false, nil)

	//获取服务器的ip端口,和一些其它信息
	xclient := client.NewXClient(serviceName, client.Failover, client.RoundRobin, d, client.DefaultOption)
	c.xc = xclient
	return c
}

func (c *ClientManager) Call(ctx context.Context, method string, args interface{}, reply interface{}) error {
	newCtx, f := context.WithTimeout(ctx, c.callTimeout)
	defer f()
	// c := s.getXClient()
	return c.xc.Call(newCtx, method, args, reply)
}
