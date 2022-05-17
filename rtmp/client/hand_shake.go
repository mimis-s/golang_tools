package rtmp

import (
	"bufio"
	"net"
	"time"
)

var (
	timeout = 5 * time.Second
)

type Conn struct {
	net.Conn
	ReadWriter *bufio.ReadWriter
}

// rtmp三次握手
// 作为客户端
func (c *Conn) ShakeAsClinet() error {
	// C0 -> 1 字节
	// C1 -> 1536 字节
	// C2 -> 1536 字节
	C0C1C2 := make([]byte, 0, 1+1536*2)
	C0 := C0C1C2[:1]
	C0C1 := C0C1C2[:1536+1]

	// rtmp版本号
	C0[0] = 3

	// C0C1
	c.Conn.SetReadDeadline(time.Now().Add(timeout))
	if _, err := c.ReadWriter.Write(C0C1); err != nil {
		return err
	}
	c.Conn.SetReadDeadline(time.Now().Add(timeout))
	if err := c.ReadWriter.Flush(); err != nil {
		return err
	}

	// S0 -> 1 字节
	// S1 -> 1536 字节
	// S2 -> 1536 字节
	S0S1S2 := make([]byte, 0, 1+1536*2)
	// S0 := S0S1S2[:1]
	// S2 := S0S1S2[1536+1:]

	S1 := S0S1S2[1 : 1536+1]
	C2 := S1

	// C2
	c.Conn.SetDeadline(time.Now().Add(timeout))
	if _, err := c.ReadWriter.Write(C2); err != nil {
		return err
	}
	return nil
}
