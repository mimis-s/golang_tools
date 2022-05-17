package clientConn

import (
	"bufio"
	"fmt"
	"io"
	"net"

	"gitee.com/mimis/golang-tool/lib/zbyte"
)

type CallBackFunc func(*ClientMsg) (*ClientMsg, error)

type ClientMsg struct {
	Tag int
	Msg []byte
}

type ClientConn struct {
	conn      net.Conn
	inStream  *bufio.Reader
	recvQueue chan *ClientMsg // 接收队列
	sendQueue chan []byte     // 发送队列
}

func NewClientConn(conn net.Conn) *ClientConn {
	return &ClientConn{
		conn:      conn,
		inStream:  bufio.NewReader(conn),
		recvQueue: make(chan *ClientMsg),
		sendQueue: make(chan []byte),
	}
}

func ReadClientMsg(inStream *bufio.Reader) (*ClientMsg, error) {

	head := make([]byte, 8)
	//ID和长度
	_, err := io.ReadFull(inStream, head)
	if err != nil {
		return nil, err
	}
	msgID := zbyte.BigEndByteToInt32(head[:4])
	msgLen := zbyte.BigEndByteToInt32(head[4:])

	// 消息体
	msgByte := make([]byte, msgLen)
	msgLenRes, err := io.ReadFull(inStream, msgByte)
	if err != nil || msgLenRes != msgLen {
		return nil, fmt.Errorf("recv client msg is err")
	}
	return &ClientMsg{
		Tag: msgID,
		Msg: msgByte,
	}, nil
}

// 解析消息
func (c *ClientConn) ReadRecvMsg() {
	for {
		packet, err := ReadClientMsg(c.inStream)
		if err != nil {
			fmt.Printf("client[%v] read msg is err:%v\n", c.conn, err)
			c.conn.Close()
			break
		}

		// 把解析出来的消息传递给处理线程
		c.recvQueue <- packet
	}
}

// 处理消息
func (c *ClientConn) DeliverRecvMsg(call CallBackFunc) {
	for {
		select {
		case msg, ok := <-c.recvQueue:
			if !ok {
				return
			}

			// 调用回调函数
			res, err := call(msg)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)

			}

			// 将消息封装起来(id + 长度 + 消息)
			buf := make([]byte, 0, 8+len(res.Msg))
			buf = append(buf, zbyte.BigEndInt32ToByte(res.Tag)...)      // id占4个字节
			buf = append(buf, zbyte.BigEndInt32ToByte(len(res.Msg))...) // 长度占4个字节
			buf = append(buf, res.Msg...)

			c.sendQueue <- buf
		}
	}
}

// 发送消息
func (c *ClientConn) WriteMsg() {
	for {
		select {
		case sendMsg, ok := <-c.sendQueue:
			if !ok {
				return
			}

			// 发送给客户端
			_, err := c.conn.Write(sendMsg)
			if err != nil {
				fmt.Printf("[net core]conn[%v] write msg error:%v\n", c.conn.RemoteAddr().String(), err)
			}
		}
	}
}
