package kcp

import "encoding/binary"

// 对seg结构进行编码
func (seg *IKCPSEG) Encode() []byte {
	data := make([]byte, 0, IKCP_OVERHEAD)
	binary.BigEndian.PutUint32(data[0:4], seg.Conv)
	data[4] = byte(seg.Cmd)
	data[5] = byte(seg.Frg)
	binary.BigEndian.PutUint16(data[6:8], uint16(seg.Wnd))
	binary.BigEndian.PutUint32(data[8:12], seg.Ts)
	binary.BigEndian.PutUint32(data[12:16], seg.Sn)
	binary.BigEndian.PutUint32(data[16:20], seg.Una)
	binary.BigEndian.PutUint32(data[20:24], seg.Len)
	return data
}
