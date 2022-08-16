package kcp

import "unsafe"

// 创建一个kcp连接实例
func CreateIKCP(conv uint32, user unsafe.Pointer) *IKCPCB {
	return &IKCPCB{
		Conv: conv, User: user,
		Snd_wnd:   IKCP_WND_SND,
		Rcv_wnd:   IKCP_WND_RCV,
		Rmt_wnd:   IKCP_WND_RCV,
		Mtu:       IKCP_MTU_DEF,
		Mss:       IKCP_MTU_DEF - IKCP_OVERHEAD,
		Buffer:    make([]byte, 0, (IKCP_MTU_DEF+IKCP_OVERHEAD)*3),
		Snd_queue: NewIQueueHead(),
		Rcv_queue: NewIQueueHead(),
		Snd_buf:   NewIQueueHead(),
		Rcv_buf:   NewIQueueHead(),
		Rx_rto:    int32(IKCP_RTO_DEF),
		Rx_minrto: int32(IKCP_RTO_MIN),
		Interval:  IKCP_INTERVAL,
		Ts_flush:  IKCP_INTERVAL,
		Ssthresh:  IKCP_THRESH_INIT,
		FastLimit: int(IKCP_FASTACK_LIMIT),
		Dead_link: IKCP_DEADLINK,
	}
}

// 设置下层协议输出函数回调
func (kcp *IKCPCB) SetOutput(output CallBackFunc) {
	kcp.Output = output
}

// 接收
func (kcp *IKCPCB) Recv(buffer []byte, buf_len int) int {
	return 0
}

// 发送
func (kcp *IKCPCB) Send(buffer []byte, buf_len int) {
}

// 更新状态,每10~100ms调用一次,current当前时间戳
func (kcp *IKCPCB) Update(current uint32) {
}

// 使用check来调用update,如果现在没有input/send操作,则可以调用update,而不是一直重复调用
func (kcp *IKCPCB) Check(current uint32) {
}

// 输入数据
func (kcp *IKCPCB) Input(data []byte, data_size int) {

}

// 刷新数据
func (kcp *IKCPCB) Flush() {
}

// 检查接收队列中下一条消息的大小
func (kcp *IKCPCB) PeekSize() int {

	return 0
}

// 修改MTU大小,默认为1400
func (kcp *IKCPCB) SetMtu(mtu int) int {
	return 0
}

// 设置发送和接收窗口的最大值,默认为sndwnd=32, rcvwnd=32
func (kcp *IKCPCB) WndSize(sndwnd int, rcvwnd int) int {
	return 0
}

// 获取等待发送的包的数量
func (kcp *IKCPCB) WaitSnd() int {
	return 0
}

// 启动快速模式, nodelay-> 0禁用,1启用, interval -> 内部更新定时器间隔,默认为100ms,
// resend -> 0禁用快速重传,1启用, nc ->0启用拥塞控制,1禁用
func (kcp *IKCPCB) SetNodelay(nodelay, interval, resend, nc int) int {
	return 0
}

// 获取conv,唯一标识
func (kcp *IKCPCB) GetConv() uint32 {
	return 0
}

// 日志(不实现)
func (kcp *IKCPCB) Log(mask int, fmt string) {
}
