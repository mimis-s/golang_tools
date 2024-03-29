package kcp

import (
	"github.com/mimis-s/golang_tools/lib"
)

// 发送方接收缓冲区还未接收的最小报文段编号una,如果比发送缓冲区sn大,那就从发送缓冲区删除
// 例:1和2通信,2没有收到的sn最小的号una如果比1中发送缓冲区的sn要大,说明1中的发送缓冲区有很多对方已经收到的消息,但是没有删除,所以要删除
func (kcp *IKCPCB) Parse_Una(una uint32) {
	for index, seg := range kcp.Snd_buf {
		if una > seg.Sn {
			// 删除已经发送的包
			kcp.Snd_buf = append(kcp.Snd_buf[:index], kcp.Snd_buf[index+1:]...)
			kcp.Nsnd_buf-- // 这个东西是c语言原版有的东西,现在先不删
		}
		// 原版在这里直接else break了, 这里我持保守意见,不加这个break
	}
}

// 修改发送缓冲区最小还未确认送达的报文段编号
func (kcp *IKCPCB) Shrink_Buf() {
	if len(kcp.Snd_buf) != 0 {
		kcp.Snd_una = kcp.Snd_buf[0].Una // 发送缓冲区最小编号的报文段
	} else {
		kcp.Snd_una = kcp.Snd_nxt // 下一个等待发送的报文端sn
	}
}

// 计算超时重传rx_rto,用到中间变量rx_srtt, rx_rttval
func (kcp *IKCPCB) Update_Ack(rtt int32) {
	rto := int32(0)
	if kcp.Rx_srtt == 0 {
		kcp.Rx_srtt = rtt
		kcp.Rx_rttval = rtt / 2
	} else {
		delta := rtt - kcp.Rx_srtt
		if delta < 0 {
			delta -= delta
		}
		kcp.Rx_rttval = (3*kcp.Rx_srtt + delta) / 4
		kcp.Rx_srtt = (7*kcp.Rx_srtt + rtt) / 8
		if kcp.Rx_srtt < 1 {
			kcp.Rx_srtt = 1
		}
	}
	rto = kcp.Rx_srtt + lib.MaxInt32(int32(kcp.Interval), 4*kcp.Rx_rttval)
	kcp.Rx_rto = lib.MinInt32(lib.MaxInt32(kcp.Rx_minrto, int32(rto)), int32(IKCP_RTO_MAX))
}

func (kcp *IKCPCB) Parse_Ack(sn uint32) {
	if sn < kcp.Snd_una || sn >= kcp.Snd_nxt {
		return
	}

	for index, seg := range kcp.Snd_buf {
		if sn == seg.Sn {
			kcp.Snd_buf = append(kcp.Snd_buf[:index], kcp.Snd_buf[index+1:]...)
			kcp.Nsnd_buf--
			break
		}
		if sn < seg.Sn {
			break
		}
	}
}

func (kcp *IKCPCB) Ack_Push(sn uint32, ts uint32) {
	// 原版这里要判断给acklist扩容,但是go里面切片不需要我们手动扩容
	kcp.AckList[kcp.AckCount*2] = sn
	kcp.AckList[kcp.AckCount*2+1] = ts
	kcp.AckCount++
}

func (kcp *IKCPCB) Parse_Data(newSeg *IKCPSEG) {
	repeat := 0
	if newSeg.Sn >= kcp.Rcv_nxt+kcp.Rcv_wnd || newSeg.Sn < kcp.Rcv_nxt {
		return
	}

	for _, seg := range kcp.Rcv_buf {
		if seg.Sn == newSeg.Sn {
			repeat = 1
			break
		}
		if newSeg.Sn > seg.Sn {
			break
		}
	}

	if repeat == 0 {
		kcp.Rcv_buf = append(kcp.Rcv_buf, *newSeg)
		kcp.Nrcv_buf++
	} else {
		return
	}

	// 将可用数据移动rcv_buf -> rcv_queue
	for index, seg := range kcp.Rcv_buf {
		if seg.Sn == kcp.Rcv_nxt && kcp.Nrcv_que < kcp.Rcv_wnd {
			kcp.Rcv_queue = append(kcp.Rcv_queue, seg)
			kcp.Rcv_buf = append(kcp.Rcv_buf[:index], kcp.Rcv_buf[index+1:]...)
			kcp.Nrcv_buf--
			kcp.Nrcv_que++
			kcp.Rcv_nxt++
		} else {
			break
		}
	}
}

func (kcp *IKCPCB) Parse_Faskack(sn, ts uint32) {
	if sn < kcp.Snd_una || sn >= kcp.Snd_nxt {
		return
	}

	for _, seg := range kcp.Snd_buf {
		if sn < seg.Sn {
			break
		} else if sn != seg.Sn {
			seg.FastACK++
		}
	}
}

/*
	--------------------四种数据类型实现-------------------------
	IKCP_CMD_ACK, IKCP_CMD_PUSH, IKCP_CMD_WASK, IKCP_CMD_WINS
*/

// ack, 当我们收到的消息类型是IKCP_CMD_ACK的时候被调用
func cmdAck(kcp *IKCPCB, seg *IKCPSEG) {
	// 如果当前时间大于等于对方发送消息时间,则更新ack,这里面包含了大量的计算,主要用于计算超时重传rx_rto
	if kcp.Current >= seg.Ts {
		kcp.Update_Ack(int32(kcp.Current - seg.Ts))
	}
	kcp.Parse_Ack(seg.Sn)
	kcp.Shrink_Buf()

}

// push
func cmdPush(kcp *IKCPCB, seg *IKCPSEG) {

}

// wask
func cmdWask(kcp *IKCPCB, seg *IKCPSEG) {

}

// wins
func cmdWins(kcp *IKCPCB, seg *IKCPSEG) {

}
