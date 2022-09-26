package kcp

import "unsafe"

// 所有类型不考虑跨平台,都是基于linux-x64开发

//=====================================================================
// KCP BASIC
//=====================================================================
const (
	IKCP_RTO_NDL       uint32 = 30  // no delay min rto
	IKCP_RTO_MIN       uint32 = 100 // normal min rto
	IKCP_RTO_DEF       uint32 = 200
	IKCP_RTO_MAX       uint32 = 60000
	IKCP_CMD_PUSH      uint32 = 81 // cmd: push data
	IKCP_CMD_ACK       uint32 = 82 // cmd: ack
	IKCP_CMD_WASK      uint32 = 83 // cmd: window probe (ask)
	IKCP_CMD_WINS      uint32 = 84 // cmd: window size (tell)
	IKCP_ASK_SEND      uint32 = 1  // need to send IKCP_CMD_WASK
	IKCP_ASK_TELL      uint32 = 2  // need to send IKCP_CMD_WINS
	IKCP_WND_SND       uint32 = 32
	IKCP_WND_RCV       uint32 = 128 // must >= max fragment size
	IKCP_MTU_DEF       uint32 = 1400
	IKCP_ACK_FAST      uint32 = 3
	IKCP_INTERVAL      uint32 = 100
	IKCP_OVERHEAD      uint32 = 24
	IKCP_DEADLINK      uint32 = 20
	IKCP_THRESH_INIT   uint32 = 2
	IKCP_THRESH_MIN    uint32 = 2
	IKCP_PROBE_INIT    uint32 = 7000   // 7 secs to probe window size
	IKCP_PROBE_LIMIT   uint32 = 120000 // up to 120 secs to probe window
	IKCP_FASTACK_LIMIT uint32 = 5      // max times to trigger fastack
)

//=====================================================================
// segment 段结构
//=====================================================================
type IKCPSEG struct {
	Conv     uint32 // conv 双方协商的唯一识别码, 收发端一致
	Cmd      uint32 // cmd 报文段类型,IKCP_CMD_PUSH, IKCP_CMD_ACK, IKCP_CMD_WASK, IKCP_CMD_WINS
	Frg      uint32 // frg 剩余包的分片数量, 这个包之后还有多少个报文属于这个包
	Wnd      uint32 // wnd 发送方剩余接收窗口大小
	Ts       uint32 // ts 发送时间戳,用来计算RTT(往返时间), 然后计算出RTO(超时重传时间)
	Sn       uint32 // sn 报文编号,唯一标识报文
	Una      uint32 // una 发送方接收缓冲区还未接收的最小报文段编号
	Len      uint32 // len 后面的数据长度
	ReSendTs uint32 // 下次超时重传的时间戳
	Rto      uint32 // rto 超时重传时间
	FastACK  uint32 // fastack 快速重传, 收到ack时计算的该分片被跳过的累计次数
	Xmit     uint32 // 该链接超时重传的总次数
	Data     []byte // data 数据
}

//---------------------------------------------------------------------
// IKCPCB 表示KCP连接,方法实现在kcp.go里面
//---------------------------------------------------------------------
type IKCPCB struct {
	// mtu 最大传输单元, mss 最大报文段大小mss=mtu-包头长度(24),state 连接状态 0连接建立,-1断开(因为是uint32所以-1为0xffffffff)
	Conv, Mtu, Mss uint32
	State          int32
	// snd_una 发送缓冲区最小还未确认送达的报文段编号,snd_nxt 下一个等待发送报文段编号,rcv_nxt下一个等待接收段编号
	Snd_una, Snd_nxt, Rcv_nxt uint32
	// ts_recent, ts_lastack未使用, ssthresh 慢启动阀值
	Ts_recent, Ts_lastack, Ssthresh uint32
	// rx_rto 超时重传时间, rx_rttval, rx_srtt, rx_minrto: 计算 rx_rto 的中间变量.
	Rx_rttval, Rx_srtt, Rx_rto, Rx_minrto int32
	// snd_wnd, rcv_wnd 发送窗口和接收窗口的大小, rmt_wnd 对端剩余接收窗口的大小.cwnd 拥塞窗口.probe 探测窗口变量(IKCP_ASK_TELL表示告知远端窗口大小。IKCP_ASK_SEND表示请求远端告知窗口大小).
	Snd_wnd, Rcv_wnd, Rmt_wnd, Cwnd, Probe uint32
	// current 当前时间,interval flush 的时间粒度,ts_flush 下次需要 flush 的时间,xmit 该链接超时重传的总次数
	Current, Interval, Ts_flush, Xmit uint32
	// nrcv_buf, nsnd_buf 接收缓冲区, 发送缓冲区的长度
	Nrcv_buf, Nsnd_buf uint32
	// nrcv_que, nsnd_que 接收队列, 发送队列的长度
	Nrcv_que, Nsnd_que uint32
	// nodelay: 是否启动快速模式,updated 是否调用过 ikcp_update
	Nodelay, Updated uint32
	// ts_probe下次窗口探测的时间戳, probe_wait 发送探测窗口消息的间隔时间
	Ts_probe, Probe_wait uint32
	// dead_link 当一个报文发送超时次数达到 dead_link 次时认为连接断开, incr 用于计算 cwnd
	Dead_link, Incr uint32
	Snd_queue       []IKCPSEG // snd_queue 发送队列
	Rcv_queue       []IKCPSEG // rcv_queue 接收队列
	Snd_buf         []IKCPSEG // 发送缓冲区
	Rcv_buf         []IKCPSEG // 接收缓冲区
	// acklist, ackcount, ackblock: ACK 列表, ACK 列表的长度和容量.
	// 待发送的 ACK 的相关信息会先存在 ACK 列表中, flush 时一并发送.
	AckList  []uint32
	AckCount uint32
	AckBlock uint32
	User     unsafe.Pointer
	// buffer flush 时用到的临时缓冲区
	Buffer []byte
	// fastresend ACK 失序 fastresend 次时触发快速重传
	FastReSend int
	// fastlimit 传输次数小于 fastlimit 的报文才会执行快速重传
	FastLimit int
	// nocwnd 是否不考虑拥塞窗口, stream 是否开启流模式, 开启后可能会合并包
	Nocwnd, Stream int
	// logmask: 用于控制日志
	Logmask int

	// output 下层协议输出函数.
	Output CallBackFunc

	// void (*writelog)(const char *log, struct IKCPCB *kcp, void *user);		日志函暂时不需要
}

type CallBackFunc func(buf []byte, buf_len int, kcp *IKCPCB, user unsafe.Pointer)
