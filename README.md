# golang工具

#### 介绍
一些golang的工具库

#### 安装教程

go get https://github.com/mimis-s/golang_tools

#### 使用说明

1.  网络库对外有两个接口:InitService(),Listen()
    InitService接口的传入参数是addr，协议(现在仅支持TCP)类型，消息回调函数
    Listen接口主要是开启网络服务，监听，收发消息

2.  rpcx库主要封装了rpcx服务器的创建，运行接口，客户端创建，调用接口，代码简单明了，
    详细使用可参考protobuf插件集成rpcx: https://github.com/mimis-s/protoc-gen-rpcx

3.  dfs封装了s3,cloud storage, minio三个分布式文件存储系统的集成api