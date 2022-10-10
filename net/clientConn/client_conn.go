package clientConn

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"

	"golang.org/x/net/websocket"
)

// 序列化方式
type Enum_SerializationMethod int

const (
	Enum_SerializationMethod_JSON  Enum_SerializationMethod = 1 // json
	Enum_SerializationMethod_PROTO Enum_SerializationMethod = 1 // protobuf
)

type CallBackFunc func(*ClientMsg) (*ClientMsg, error)

type ClientMsg struct {
	Tag int
	Msg []byte
}

type ConnInterface interface {
}

type ClientConn struct {
	conn                     net.Conn
	inStream                 *bufio.Reader
	recvQueue                chan *ClientMsg // 接收队列
	sendQueue                chan []byte     // 发送队列
	Enum_SerializationMethod                 // 序列化方式
}

func NewClientConn(conn net.Conn, method Enum_SerializationMethod) *ClientConn {
	return &ClientConn{
		conn:                     conn,
		inStream:                 bufio.NewReader(conn),
		recvQueue:                make(chan *ClientMsg),
		sendQueue:                make(chan []byte),
		Enum_SerializationMethod: method,
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

// json
func (c *ClientConn) ReadJsonClientMsg() (*ClientMsg, error) {

	conn := c.conn.(*websocket.Conn)
	msg := ""
	err := websocket.Message.Receive(conn, &msg)
	if err != nil {
		// 服务器主动断开
		c.conn.Close()
		return nil, err
	}

	req := make(map[string]string)
	err = json.Unmarshal([]byte(msg), &req)
	if err != nil {
		err = fmt.Errorf("unmarshal request message error:%v", err)
		c.conn.Write([]byte(err.Error()))
		return nil, err
	}
	msgID, find := req["msg_id"]
	if !find {
		err = fmt.Errorf("not found msg id:%v", msg)
		c.conn.Write([]byte(err.Error()))
		return nil, err
	}

	tag, err := strconv.Atoi(msgID)
	if err != nil {
		err = fmt.Errorf("msg id is not integer:%v", msg)
		c.conn.Write([]byte(err.Error()))
		return nil, err
	}

	payload, find := req["payload"]
	if !find {
		err = fmt.Errorf("not found msg:%v", msg)
		c.conn.Write([]byte(err.Error()))
		return nil, err
	}

	return &ClientMsg{
		Tag: tag,
		Msg: []byte(payload),
	}, nil
}

func (c *ClientConn) WriteJsonClientMsg(tag int, msg []byte) ([]byte, error) {
	res := make(map[string]interface{})
	err := json.Unmarshal(msg, &res)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error:%v", err)
	}
	buf := map[string]interface{}{
		"msg_id":  tag,
		"payload": res,
	}
	sendMsg, err := json.Marshal(&buf)
	if err != nil {
		return nil, fmt.Errorf("json marshal error:%v", err)
	}
	return sendMsg, nil
}

// 解析消息
func (c *ClientConn) ReadRecvMsg() {
	for {
		var err error
		var packet *ClientMsg

		if c.Enum_SerializationMethod == Enum_SerializationMethod_JSON {
			packet, err = c.ReadJsonClientMsg()
			if err != nil {
				fmt.Printf("client[%v] read msg is err:%v\n", c.conn, err)
				c.conn.Close()
				break
			}
		} else {
			packet, err = c.ReadBytesClientMsg()
			if err != nil {
				fmt.Printf("client[%v] read msg is err:%v\n", c.conn, err)
				c.conn.Close()
				break
			}
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
				continue
			}

			var buf []byte
			if c.Enum_SerializationMethod == Enum_SerializationMethod_PROTO {
				buf, err = c.WriteBytesClientMsg(res.Tag, res.Msg)
			} else {
				buf, err = c.WriteJsonClientMsg(res.Tag, res.Msg)
			}
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
				fmt.Printf("[net core]conn[%v] write msg error:%v\n", c.conn.RemoteAddr().String(), err)
			}
		}
	}
}
