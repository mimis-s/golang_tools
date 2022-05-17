package rtmp

import (
	"fmt"
	"io"
	"time"

	"gitee.com/mimis/golang-tool/lib/zbyte"
)

var (
	timeout = 5 * time.Second
)

// rtmp三次握手
// 作为服务器
func (c *Conn) ShakeAsService() error {
	C0C1C2 := make([]byte, 1+1536*2)
	C0 := C0C1C2[:1]
	C1 := C0C1C2[1 : 1536+1]
	C0C1 := C0C1C2[:1536+1]
	C2 := C0C1C2[1536+1:]

	S0S1S2 := make([]byte, 1+1536*2)
	S0 := S0S1S2[:1]
	S1 := S0S1S2[1 : 1536+1]
	// S0S1 := S0S1S2[:1536+1]
	S2 := S0S1S2[1536+1:]

	// < C0C1
	c.Conn.SetDeadline(time.Now().Add(timeout))
	if _, err := io.ReadFull(c.ReadWriter, C0C1); err != nil {
		return err
	}
	c.Conn.SetDeadline(time.Now().Add(timeout))
	if C0[0] != 3 {
		c.Conn.Close()
		return fmt.Errorf("rtmp: handshake version=%d invalid", C0[0])
	}

	S0[0] = 3

	// clitime := zbyte.ByteToInt(C1[0:4])
	// srvtime := clitime
	// srvver := uint32(0x0d0e0a0d)
	cliver := zbyte.BigEndByteToInt32(C1[4:8])

	if cliver != 0 {
		return fmt.Errorf("rtmp: handshake C1 4->8byte != 0")
	} else {
		copy(S1, C2)
		copy(S2, C1)
	}

	// > S0S1S2
	c.Conn.SetDeadline(time.Now().Add(timeout))
	if _, err := c.ReadWriter.Write(S0S1S2); err != nil {
		return err
	}
	c.Conn.SetDeadline(time.Now().Add(timeout))
	if err := c.ReadWriter.Flush(); err != nil {
		return err
	}

	// < C2
	c.Conn.SetDeadline(time.Now().Add(timeout))
	if _, err := io.ReadFull(c.ReadWriter, C2); err != nil {
		return err
	}
	c.Conn.SetDeadline(time.Time{})
	return nil
}
