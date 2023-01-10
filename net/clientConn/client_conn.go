package clientConn

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type ClientMsg struct {
	Tag int
	Msg []byte
}

type ClientConn_tcp struct {
	conn      net.Conn
	inStream  *bufio.Reader
	recvQueue chan *ClientMsg // 接收队列
	sendQueue chan []byte     // 发送队列
}

func NewClientConn(conn net.Conn) *ClientConn_tcp {
	return &ClientConn_tcp{
		conn:      conn,
		inStream:  bufio.NewReader(conn),
		recvQueue: make(chan *ClientMsg),
		sendQueue: make(chan []byte),
	}
}

func (c *ClientConn_tcp) GetConnType() ClientConn_Enum {
	return ClientConn_TCP_Enum
}

func (c *ClientConn_tcp) GetIP() string {
	return c.conn.RemoteAddr().String()
}

func (c *ClientConn_tcp) GetConn() interface{} {
	return c.conn
}

// 字节流
func (c *ClientConn_tcp) uncode() (*ClientMsg, error) {

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

func (c *ClientConn_tcp) decode(tag int, msg []byte) ([]byte, error) {
	// 将消息封装起来(id + 长度 + 消息)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[:4], uint32(tag))      // id占4个字节
	binary.BigEndian.PutUint32(buf[4:], uint32(len(msg))) // 长度占4个字节
	buf = append(buf, msg...)
	return buf, nil
}

// 外部调用发送消息
func (c *ClientConn_tcp) SendMsg(res *ClientMsg) error {
	var buf []byte
	buf, err := c.decode(res.Tag, res.Msg)
	if err != nil {
		errStr := fmt.Sprintf("client msg is err:%v\n", err)
		return fmt.Errorf(errStr)
	}

	c.sendQueue <- buf
	return nil
}

// 解析消息
func (c *ClientConn_tcp) ReadRecvMsg(session ClientSession) {
	for {
		var err error
		var packet *ClientMsg
		packet, err = c.uncode()
		if err != nil {
			fmt.Printf("client[%v] read msg is err:%v\n", c.conn, err)
			c.conn.Close()
			session.DisConnectCallBack()
			break
		}

		// 把解析出来的消息传递给处理线程
		c.recvQueue <- packet
	}
}

// 处理消息
func (c *ClientConn_tcp) DeliverRecvMsg(session ClientSession) {
	for {
		select {
		case msg, ok := <-c.recvQueue:
			if !ok {
				return
			}

			// 调用回调函数
			res, err := session.RequestCallBack(msg)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)
				continue
			}

			var buf []byte
			buf, err = c.decode(res.Tag, res.Msg)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)
				continue
			}

			c.sendQueue <- buf
		}
	}
}

// 把消息发送给客户端
func (c *ClientConn_tcp) WriteMsg() {
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
