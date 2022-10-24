package clientConn

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type ClientSession interface {
	ConnectCallBack()                               // 客户端连接回调
	RequestCallBack(*ClientMsg) (*ClientMsg, error) // 消息处理的回调
	DisConnectCallBack()                            // 客户端断开连接回调
}

// type CallBackFunc func(*ClientMsg) (*ClientMsg, error)

type ClientMsg struct {
	Tag int
	Msg []byte
}

type ClientConn struct {
	conn      net.Conn
	inStream  *bufio.Reader
	session   ClientSession
	recvQueue chan *ClientMsg // 接收队列
	sendQueue chan []byte     // 发送队列
}

func NewClientConn(conn net.Conn, session ClientSession) *ClientConn {
	return &ClientConn{
		conn:      conn,
		inStream:  bufio.NewReader(conn),
		session:   session,
		recvQueue: make(chan *ClientMsg),
		sendQueue: make(chan []byte),
	}
}

// 字节流
func (c *ClientConn) ReadBytesClientMsg() (*ClientMsg, error) {

	head := make([]byte, 8)
	//ID和长度
	_, err := io.ReadFull(c.inStream, head)
	if err != nil {
		return nil, err
	}
	msgID := int(binary.BigEndian.Uint32(head[:4]))
	msgLen := int(binary.BigEndian.Uint32(head[4:]))

	// 消息体
	msgByte := make([]byte, msgLen)
	msgLenRes, err := io.ReadFull(c.inStream, msgByte)
	if err != nil || msgLenRes != msgLen {
		return nil, fmt.Errorf("recv client msg is err")
	}
	return &ClientMsg{
		Tag: msgID,
		Msg: msgByte,
	}, nil
}

func (c *ClientConn) WriteBytesClientMsg(tag int, msg []byte) ([]byte, error) {
	// 将消息封装起来(id + 长度 + 消息)
	buf := make([]byte, 0, 8+len(msg))
	binary.BigEndian.PutUint32(buf[:4], uint32(tag))      // id占4个字节
	binary.BigEndian.PutUint32(buf[4:], uint32(len(msg))) // 长度占4个字节
	buf = append(buf, msg...)
	return buf, nil
}

// 解析消息
func (c *ClientConn) ReadRecvMsg() {
	for {
		var err error
		var packet *ClientMsg
		packet, err = c.ReadBytesClientMsg()
		if err != nil {
			fmt.Printf("client[%v] read msg is err:%v\n", c.conn, err)
			c.conn.Close()
			c.session.DisConnectCallBack()
			break
		}

		// 把解析出来的消息传递给处理线程
		c.recvQueue <- packet
	}
}

// 处理消息
func (c *ClientConn) DeliverRecvMsg() {
	for {
		select {
		case msg, ok := <-c.recvQueue:
			if !ok {
				return
			}

			// 调用回调函数
			res, err := c.session.RequestCallBack(msg)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)
				continue
			}

			var buf []byte
			buf, err = c.WriteBytesClientMsg(res.Tag, res.Msg)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)
				continue
			}

			c.sendQueue <- buf
		}
	}
}

// 把消息发送给客户端
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
				fmt.Printf("[net write msg error:%v\n", err)
			}
		}
	}
}
