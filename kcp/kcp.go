package kcp

import (
	"encoding/binary"
	"unsafe"

	"gitee.com/mimis/golang-tool/lib"
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

// 发送, 将要发送的数据转换成KCP格式, 添加到snd_queue,当数据大于一个MSS(最大分片大小)就对数据进行分片
// 分片个数不能大于255
func (kcp *IKCPCB) Send(buffer []byte) int {
	if len(buffer) <= 0 || kcp.Mss <= 0 {
		return -1
	}

	// 计算分片数量
	splitNum := 1
	if len(buffer) > int(kcp.Mss) {
		splitNum = len(buffer) / int(kcp.Mss)
	}

	if splitNum > 255 {
		return -2
	}

	// 组装seg结构,加入snd_buf
	buffer_2 := buffer
	for i := 0; i < splitNum; i++ {
		splitLen := lib.MinInt(len(buffer_2), int(kcp.Mss))
		seg := IKCPSEG{
			Len:  uint32(splitLen),
			Frg:  uint32(splitNum - i - 1), // 分片的序号, 从大到小递减
			Data: buffer_2[:splitLen],
		}

		kcp.Snd_queue = append(kcp.Snd_queue, seg)
		kcp.Nsnd_que++ // 这个可以不用

		buffer_2 = buffer_2[splitLen:]
	}
	return 0
}

// 更新状态,每10~100ms调用一次,current当前时间戳
func (kcp *IKCPCB) Update(current uint32) {
}

// 使用check来调用update,如果现在没有input/send操作,则可以调用update,而不是一直重复调用
func (kcp *IKCPCB) Check(current uint32) {
}

// 读取下层UDP输入数据,传入KCP结构,如果是数据就放入接收缓冲区,如果是ack就在发送缓冲区标记对应报文段已送达
func (kcp *IKCPCB) Input(data []byte) int {

	prev_una := kcp.Snd_una // prev_una发送端缓冲区最小未接收una
	maxack := uint32(0)     // 接收到最大的一个ack序列号sn
	latest_ts := uint32(0)  // 这个ack消息对应的发送时间戳ts
	flag := 0

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

		if len(data) < int(seg.Len) || seg.Len < uint32(0) {
			// 剩下的可读取长度小于设定的长度
			return -2
		}

		// 如果不是push, ack, wask, wins这四个类型,则返回-3
		if seg.Cmd != IKCP_CMD_ACK && seg.Cmd != IKCP_CMD_PUSH &&
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
		// mapCmd_Type[seg.Cmd](kcp, seg)
		if seg.Cmd == IKCP_CMD_ACK {
			// 如果当前时间大于等于对方发送消息时间,则更新ack,这里面包含了大量的计算,主要用于计算超时重传rx_rto
			if kcp.Current >= seg.Ts {
				kcp.Update_Ack(int32(kcp.Current - seg.Ts))
			}
			kcp.Parse_Ack(seg.Sn)
			kcp.Shrink_Buf()
			if flag == 0 {
				flag = 1
				maxack = seg.Sn
				latest_ts = seg.Ts
			} else {
				if seg.Sn > maxack {
					if seg.Ts > latest_ts {
						maxack = seg.Sn
						latest_ts = seg.Ts
					}
				}
			}
			// 日志打印
		} else if seg.Cmd == IKCP_CMD_PUSH {
			// 日志打印
			if seg.Sn < kcp.Rcv_nxt+kcp.Rcv_wnd {
				// 加入acklist
				kcp.Ack_Push(seg.Sn, seg.Ts)
				if seg.Sn >= kcp.Rcv_nxt {
					if seg.Len > 0 {
						seg.Data = data
					}
					kcp.Parse_Data(seg)
				}
			}
		} else if seg.Cmd == IKCP_CMD_WASK {
			// 窗口探测消息,告诉远端我的窗口大小
			kcp.Probe |= IKCP_ASK_TELL
		} else if seg.Cmd == IKCP_CMD_WINS {
			// 什么事情也不做
		} else {
			return -3
		}

		// 读取数据往后挪动
		data = data[seg.Len:]
	}

	if flag != 0 {
		kcp.Parse_Faskack(maxack, latest_ts)
	}

	if kcp.Snd_una > prev_una {
		if kcp.Cwnd < kcp.Rmt_wnd {
			mss := kcp.Mss
			if kcp.Cwnd < kcp.Ssthresh {
				kcp.Cwnd++
				kcp.Incr += mss
			} else {
				if kcp.Incr < mss {
					kcp.Incr = mss
				}
				kcp.Incr += (mss*mss)/kcp.Incr + (mss / 16)
				if (kcp.Cwnd+1)*mss <= kcp.Incr {
					temp := uint32(1)
					if mss > 0 {
						temp = mss
					}
					kcp.Cwnd = (kcp.Incr + mss - 1) / temp
				}
			}
			if kcp.Cwnd > kcp.Rmt_wnd {
				kcp.Cwnd = kcp.Rmt_wnd
				kcp.Incr = kcp.Rmt_wnd * mss
			}
		}
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

// 刷新数据, 将snd_buf中的数据通过下层UDP发送出去,有以下四种情况:
// 发送ack, 发送探测窗口消息, 计算拥塞窗口大小, 将消息从snd_queue转移到snd_buf
func (kcp *IKCPCB) Flush() {

	// 判断update是否被调用
	if kcp.Updated == 0 {
		return
	}

	// 构造一个ack seg, 告诉对方自己的剩余窗口大小和还没有接收的最小sn
	seg := IKCPSEG{
		Conv: kcp.Conv,
		Cmd:  IKCP_CMD_ACK,                               // ack报文
		Wnd:  lib.MaxUint32(kcp.Rcv_wnd-kcp.Nrcv_que, 0), // 接收窗口-接收队列 = 剩余接收窗口大小
		Una:  kcp.Rcv_nxt,
	}

	// flush ack, 因为ack消息没有data,固定24byte,所以可以把多个ack拼接起来一起发送
	outputBuffer := make([]byte, 0)
	for i := 0; i < int(kcp.AckCount); i++ {

		if len(outputBuffer)+int(IKCP_OVERHEAD) > int(kcp.Mtu) {
			// 当ack序列的长度+24(一个ack长度) > mtu时发送ack序列
			// mss = mtu - 24的头,这里因为mss报文段没有数据,mtu应该是24
			kcp.Output(outputBuffer, len(outputBuffer), kcp, kcp.User)
			// 又重新初始化要发送的ack序列
			outputBuffer = make([]byte, 0)
		}

		// 这里可以得到acklist是sn和ts交替的数组
		seg.Sn = kcp.AckList[i*2]
		seg.Ts = kcp.AckList[i*2+1]

		outputBuffer = append(outputBuffer, seg.Encode()...)
	}
	kcp.AckCount = 0

	// 发送探测窗口消息
	// 设置下次探测时间戳和间隔时间
	if kcp.Rmt_wnd == 0 { // 对端接收窗口大小为0
		if kcp.Probe_wait == 0 { // 探测窗口的时间间隔为0
			kcp.Probe_wait = IKCP_PROBE_INIT // 设置探测窗口的时间间隔
			// 设置下次探测窗口的时间戳 = 当前时间 + 等待时间间隔
			kcp.Ts_probe = kcp.Current + kcp.Probe_wait
		} else {
			if kcp.Current >= kcp.Ts_probe {
				if kcp.Probe_wait < IKCP_PROBE_INIT {
					kcp.Probe_wait = IKCP_PROBE_INIT
				}
				kcp.Probe_wait += kcp.Probe_wait / 2 // 时间间隔变成1.5倍
				if kcp.Probe_wait > IKCP_PROBE_LIMIT {
					kcp.Probe_wait = IKCP_PROBE_LIMIT // 不能超过最大的时间间隔阈值
				}
				kcp.Ts_probe = kcp.Current + kcp.Probe_wait
				kcp.Probe |= IKCP_ASK_SEND // 请求对端告知窗口大小
			}
		}
	} else {
		kcp.Ts_probe = 0
		kcp.Probe_wait = 0
	}

	// 发送探测窗口消息
	if kcp.Probe == IKCP_ASK_SEND || kcp.Probe == IKCP_ASK_TELL {
		if kcp.Probe == IKCP_ASK_SEND {
			seg.Cmd = IKCP_CMD_WASK
		} else {
			seg.Cmd = IKCP_CMD_WINS
		}
		// 这里源码用前面的outputAckBuffer序列继续做操作, 很巧妙的把上面剩下的一个seg用到了现在的逻辑里面
		if len(outputBuffer)+int(IKCP_OVERHEAD) > int(kcp.Mtu) {
			// 当ack序列的长度+24(一个ack长度) > mtu时发送ack序列
			// mss = mtu - 24的头,这里因为mss报文段没有数据,mtu应该是24
			kcp.Output(outputBuffer, len(outputBuffer), kcp, kcp.User)
			// 又重新初始化要发送的ack序列
			outputBuffer = make([]byte, 0)
		}
		outputBuffer = append(outputBuffer, seg.Encode()...)
	}
	kcp.Probe = 0 // 格式化探测窗口变量

	// 计算拥塞窗口大小
	cwnd := lib.MinUInt32(kcp.Rmt_wnd, kcp.Snd_wnd) // 发送窗口和远端接收窗口取最小值
	if kcp.Nocwnd == 0 {                            // 考虑拥塞控制
		cwnd = lib.MinUInt32(kcp.Cwnd, cwnd) // 拥塞窗口和上面得到的值取最小值
	}

	// 将数据从snd_queue移动到snd_buf中
	// 滑动窗口的起点应该是snd_una(最小还未确认送达的sn),而snd_nxt(下一个等待发送的sn)必须满足不能超过滑动窗口的范围
	// 滑动窗口是用来控制发送速率的
	for kcp.Snd_nxt < kcp.Snd_una+cwnd {
		if len(kcp.Snd_queue) == 0 {
			// 没有消息需要发送
			break
		}
		newseg := kcp.Snd_queue[0]
		newseg.Conv = kcp.Conv          // 唯一会话id
		newseg.Cmd = IKCP_CMD_PUSH      // 消息类型为数据类型
		newseg.Wnd = seg.Wnd            // 发送方的接受窗口大小
		newseg.Ts = kcp.Current         // 发送的时间戳
		newseg.Sn = kcp.Snd_nxt         // 当前消息的sn
		newseg.Una = kcp.Rcv_nxt        // 发送方还未接收的最小sn等于下一个等待接受的sn
		newseg.ReSendTs = kcp.Current   // 下次超时重传的时间戳
		newseg.Rto = uint32(kcp.Rx_rto) // 超时重传时间
		newseg.FastACK = 0              // 收到ack时计算的该分片被跳过的累计次数
		newseg.Xmit = 0                 // 发送分片的次数,发送一次+1

		kcp.Snd_buf = append(kcp.Snd_buf, newseg)
		kcp.Snd_queue = kcp.Snd_queue[1:]

		// 关于长度的参数都可以删除
		kcp.Nsnd_que--
		kcp.Nsnd_buf++
		kcp.Snd_nxt++
	}

	// 消息失序次数
	resent := 0xffffffff
	if kcp.FastReSend > 0 {
		resent = kcp.FastReSend
	}
	rtomin := 0
	if kcp.Nodelay == 0 { // 如果没有启用了快速模式,超时重传时间/8
		rtomin = int(kcp.Rx_rto) >> 3
	}

	// 记录发生了超时重传
	bLost := false

	change := false

	// 检查缓存队列(snd_buf)当前需要发送的数据(新的数据和重传的数据)
	for _, segment := range kcp.Snd_buf {
		// 判断当前数据需不需要发送
		needSend := false
		if segment.Xmit == 0 {
			needSend = true
			segment.Xmit++                   // 发送分片的次数++
			segment.Rto = uint32(kcp.Rx_rto) // 当前分片的超时时间戳
			// 下面我理解为分片下次超时重传的时间，和启用快速模式也有关系
			segment.ReSendTs = kcp.Current + segment.Rto + uint32(rtomin)
		} else if kcp.Current >= segment.ReSendTs {
			// 当前时间已经超过seg的下次重传时间了
			needSend = true
			segment.Xmit++ // 超时次数+1
			kcp.Xmit++
			if kcp.Nodelay == 0 {
				// 不启动快速重传, 每次重传之后rto时间是之前的2倍
				segment.Rto += lib.MaxUint32(segment.Rto, uint32(kcp.Rx_rto))
			} else {
				// 启动快速重传, rto时间变成1.5倍
				step := kcp.Rx_rto
				if kcp.Nodelay < 2 {
					step = int32(segment.Rto)
				}
				segment.Rto += uint32(step) / 2
			}
			// 更新下一次重传的时间戳
			segment.ReSendTs = kcp.Current + segment.Rto
			bLost = true
		} else if int(segment.FastACK) >= resent {
			// 该分片被跳过的次数超过重传次数
			if segment.Xmit <= uint32(kcp.FastLimit) || kcp.FastLimit <= 0 {
				needSend = true
				segment.Xmit++
				segment.FastACK = 0
				segment.ReSendTs = kcp.Current + segment.Rto
				change = true
			}
		}

		if needSend {
			segment.Ts = kcp.Current
			segment.Wnd = seg.Wnd     // 剩余接收窗口大小
			segment.Una = kcp.Rcv_nxt // 待接收消息序号

			need := IKCP_OVERHEAD + segment.Len

			if len(outputBuffer)+int(need) > int(kcp.Mtu) {
				kcp.Output(outputBuffer, len(outputBuffer), kcp, kcp.User)
				outputBuffer = make([]byte, 0)
			}
			// 序列化头
			outputBuffer = append(outputBuffer, segment.Encode()...)
			// 序列化数据
			if segment.Len > 0 {
				outputBuffer = append(outputBuffer, segment.Data...)
			}

			if segment.Xmit >= kcp.Dead_link {
				kcp.State = -1
			}
		}
	}

	// 刷新重传seg
	if len(outputBuffer) > 0 {
		kcp.Output(outputBuffer, len(outputBuffer), kcp, kcp.User)
	}

	// 更新ssthresh(慢启动阈值)
	if change {
		// 当发生过快速重传的时候,将慢启动阈值调整为当前发送窗口的一半
		inflight := kcp.Snd_nxt - kcp.Snd_una
		kcp.Ssthresh = inflight / 2
		if kcp.Ssthresh < IKCP_THRESH_MIN {
			kcp.Ssthresh = IKCP_THRESH_MIN
		}
		// 拥塞窗口大小等于拥塞窗口阈值+触发快速重传的次数
		kcp.Cwnd = kcp.Ssthresh + uint32(resent)
		kcp.Incr = kcp.Cwnd * kcp.Mss
	}

	if bLost {
		// 如果发生了超时重传,丢包之后窗口大小减半
		kcp.Ssthresh = cwnd / 2
		if kcp.Ssthresh < IKCP_THRESH_MIN {
			kcp.Ssthresh = IKCP_THRESH_MIN
		}
		kcp.Cwnd = 1
		kcp.Incr = kcp.Mss
	}

	if kcp.Cwnd < 1 {
		kcp.Cwnd = 1
		kcp.Incr = kcp.Mss
	}

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

// 启动快速重传模式, nodelay-> 0禁用,1启用, interval -> 内部更新定时器间隔,默认为100ms,
// resend -> 快速重传的次数阈值, nc ->0启用拥塞控制,1禁用
func (kcp *IKCPCB) SetNodelay(nodelay, interval, resend, nc int) int {
	// 源码这里对于参数都进行了>=0判断,是怕传入负数参数,我们这里直接统一判断如果有一个负数,那么就直接返回-1
	if nodelay < 0 || interval < 0 || resend < 0 || nc < 0 {
		return -1
	}
	// 启用快速重传
	kcp.Nodelay = uint32(nodelay)
	if nodelay > 0 {
		kcp.Rx_minrto = int32(IKCP_RTO_NDL) // 快速重传最小超时时间
	} else {
		kcp.Rx_minrto = int32(IKCP_RTO_MIN) // 正常最小超时时间
	}

	// 内部flush刷新时间,[10, 5000]
	kcp.Interval = uint32(lib.MinInt(lib.MaxInt(nodelay, 10), 5000))
	kcp.FastReSend = resend
	kcp.Nocwnd = nc // 是否启用拥塞控制
	return 0
}

// 获取conv,唯一标识
func (kcp *IKCPCB) GetConv() uint32 {
	return 0
}

// 日志(不实现)
func (kcp *IKCPCB) Log(mask int, fmt string) {
}
