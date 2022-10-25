package clientConn

type ClientConn interface {
	GetIP() string
	GetConn() interface{}
}

type ClientSession interface {
	GetClientConn() ClientConn
	ConnectCallBack()                               // 客户端连接回调
	RequestCallBack(*ClientMsg) (*ClientMsg, error) // 消息处理的回调, 把conn连接也传入进去
	DisConnectCallBack()                            // 客户端断开连接回调
}
