package kcp

import (
	"encoding/binary"
	"unsafe"
)

type cmdFunc func(*IKCPCB, *IKCPSEG)

var mapCmd_Type = map[uint32]cmdFunc{
	IKCP_CMD_ACK:  cmdAck,
	IKCP_CMD_PUSH: cmdPush,
	IKCP_CMD_WASK: cmdWask,
	IKCP_CMD_WINS: cmdWins,
}

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
		Snd_queue: make([]IKCPSEG, 0),
		Rcv_queue: make([]IKCPSEG, 0),
		Snd_buf:   make([]IKCPSEG, 0),
		Rcv_buf:   make([]IKCPSEG, 0),
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

// 读取下层UDP输入数据,传入KCP结构,如果是数据就放入接收缓冲区,如果是ack就在发送缓冲区标记对应报文段已送达
func (kcp *IKCPCB) Input(data []byte) int {

	// prev_una := kcp.Snd_una // prev_una发送端缓冲区最小未接收una
	// maxack := uint32(0)     // 接收到最大的一个ack序列号sn
	// latest_ts := uint32(0)  // 这个ack消息对应的发送时间戳ts

	dataSize := len(data)
	// 判断数据长度是否合法
	if data == nil || dataSize < int(IKCP_OVERHEAD) {
		return -1
	}

	for {
		seg := &IKCPSEG{}
		if dataSize < int(IKCP_OVERHEAD) {
			// 24是seg结构能接收的最小长度
			break
		}
		seg.Conv = binary.BigEndian.Uint32(data)
		if seg.Conv != kcp.Conv {
			// 识别码不一致
			return -1
		}
		seg.Cmd = uint32(data[4])
		seg.Frg = uint32(data[5])
		seg.Wnd = uint32(binary.BigEndian.Uint16(data[6:8]))
		seg.Ts = binary.BigEndian.Uint32(data[8:12])
		seg.Sn = binary.BigEndian.Uint32(data[12:16])
		seg.Una = binary.BigEndian.Uint32(data[16:20])
		seg.Len = binary.BigEndian.Uint32(data[20:24])

		data = data[IKCP_OVERHEAD:]

		if len(data) < int(seg.Len) || seg.Len < 0 {
			// 剩下的可读取长度小于设定的长度
			return -2
		}

		// 如果不是push, ack, wask, wins这四个类型,则返回-3
		if seg.Cmd != IKCP_CMD_ACK && seg.Cmd != IKCP_CMD_ACK &&
			seg.Cmd != IKCP_CMD_WASK && seg.Cmd != IKCP_CMD_WINS {
			return -3
		}

		// 对端剩余接收窗口大小 = 发送方剩余接收窗口大小
		kcp.Rmt_wnd = seg.Wnd

		// 删除发送缓冲区小于已经接收的最小sn的段
		kcp.Parse_Una(seg.Una)

		// 更新发送缓冲区最小还未接收的报文段编号
		kcp.Shrink_Buf()

		// 判断四种接收报文类型(这里把原本的if else判断改为了map映射)
		mapCmd_Type[seg.Cmd](kcp, seg)
	}

	// 解析data数据到IKCPSEG结构
	// 拿到发送端缓冲区最小未接收una,检查snd_buf里面sn序列号小于una的,说明已经确认接收，删除对应seg
	// 把snd_buf发送缓冲区最后一个seg的sn赋值给发送汉冲去最小未接收una

	// 判断四种KCP报文段类型,ACK,PUSH,WASK,WINS执行不同的操作,下面依次来看每个类型的处理

	// IKCP_CMD_ACK：
	// 如果kcp当前时间比接收到的ts大，则计算RTT和RTO，把snd_buf里面和当前sn相同的seg删除
	// 把snd_buf发送缓冲区最后一个seg的sn赋值给发送汉冲去最小未接收una
	// 更新maxack=sn, latest_ts = ts

	// IKCP_CMD_PUSH:
	// 判断收到的sn比kcp的rcv_nxt下一个等待接收sn+rcv_wnd接收窗口小的时候继续下列操作
	// 这里做了滑动窗口的判断,以rcv_nxt为起点,对比sn是否超过了滑动窗口限制,超过限制的数据就会被丢弃
	// 然后新增一个ack，如果acklist容量不够，还会对ack列表进行两倍扩容
	// 判断sn必须是>=rcv_nxt的，这里可以确保对方重复发送的时候我们可以检查丢弃这个包
	// 判断sn是否与rcv_buf里面已经接收的seg相等,相等就丢弃,不相等初始化seg结构,然后初始化seg里面的node结构
	// 还要判断sn在rcv_buf应该插入的位置顺序,插入进去,然后把rcv_buf里面可用数据移动到rcv_queue里面

	// IKCP_CMD_WASK:
	// 发送控制报文,这里制作了一个kcp->probe |= IKCP_ASK_TELL操作, 之后再议

	// IKCP_CMD_WINS: 没有做任何事情

	// 结束循环读取之后判断是否需要快速重传
	return 0
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
