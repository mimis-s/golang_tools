package service

import (
	"context"
	"time"

	"github.com/rpcxio/rpcx-etcd/serverplugin"
	"github.com/smallnest/rpcx/server"
)

var (
	basePath string = "/zhangbin/rpcx"
)

type ServerManage struct {
	Addr           string // 服务器暴露出来的地址
	ListenAddr     string
	ServiceHandler *server.Server
}

func New(addr string, etcdAddr []string, etcdBasePath string, listenAddr string) (*ServerManage, error) {
	// 修改etcd默认路径
	if etcdBasePath != "" {
		basePath = etcdBasePath
	}

	// go http.ListenAndServe(addr, nil)

	s := server.NewServer()

	// 注册etcd
	err := RegisterPlugin(s, addr, etcdAddr)
	if err != nil {
		return nil, err
	}

	return &ServerManage{
		Addr:           addr,
		ServiceHandler: s,
		ListenAddr:     listenAddr,
	}, nil
}

// 注册rpc调用
func (s *ServerManage) RegisterOneService(serverName string, handler interface{}) error {
	return s.ServiceHandler.RegisterName(serverName, handler, "")
}

// 运行rpcx
func (s *ServerManage) Run() error {
	return s.ServiceHandler.Serve("tcp", s.ListenAddr)
}

func (s *ServerManage) Stop() {
	ctx, f := context.WithTimeout(context.Background(), time.Second*5)
	defer f()
	s.ServiceHandler.Shutdown(ctx)
}

func RegisterPlugin(s *server.Server, addr string, etcdAddr []string) error {
	r := &serverplugin.EtcdV3RegisterPlugin{
		ServiceAddress: "tcp@" + addr,
		EtcdServers:    etcdAddr,
		BasePath:       basePath,
		UpdateInterval: time.Minute,
	}

	err := r.Start()
	if err != nil {
		return err
	}
	s.Plugins.Add(r)
	return nil
}
