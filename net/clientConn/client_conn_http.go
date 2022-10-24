package clientConn

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gorilla/websocket"
)

type ClientConn_http struct {
	conn      *websocket.Conn
	session   ClientSession
	recvQueue chan *ClientMsg // 接收队列
	sendQueue chan []byte     // 发送队列
}

func NewClientConn_http(conn *websocket.Conn, session ClientSession) *ClientConn_http {
	return &ClientConn_http{
		conn:      conn,
		session:   session,
		recvQueue: make(chan *ClientMsg),
		sendQueue: make(chan []byte),
	}
}

// json
func (c *ClientConn_http) ReadJsonClientMsg() (*ClientMsg, error) {
	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		errStr := fmt.Sprintf("read message is err:%v", err)
		fmt.Println(errStr)
		c.session.DisConnectCallBack()
		// 服务器主动断开
		c.conn.Close()

		return nil, fmt.Errorf(errStr)
	}

	req := make(map[string]string)
	err = json.Unmarshal(msg, &req)
	if err != nil {
		err = fmt.Errorf("unmarshal request message error:%v", err)
		c.conn.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
		return nil, err
	}
	msgID, find := req["msg_id"]
	if !find {
		err = fmt.Errorf("not found msg id:%v", msg)
		c.conn.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
		return nil, err
	}

	tag, err := strconv.Atoi(msgID)
	if err != nil {
		err = fmt.Errorf("msg id is not integer:%v", msg)
		c.conn.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
		return nil, err
	}

	payload, find := req["payload"]
	if !find {
		err = fmt.Errorf("not found msg:%v", msg)
		c.conn.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
		return nil, err
	}

	return &ClientMsg{
		Tag: tag,
		Msg: []byte(payload),
	}, nil
}

func (c *ClientConn_http) WriteJsonClientMsg(tag int, msg []byte) ([]byte, error) {
	res := make(map[string]interface{})
	if len(msg) != 0 {
		err := json.Unmarshal(msg, &res)
		if err != nil {
			return nil, fmt.Errorf("json unmarshal error:%v", err)
		}
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
func (c *ClientConn_http) ReadRecvMsg_http() {
	defer func() {
		c.session.DisConnectCallBack()
		c.conn.Close()
	}()
	for {
		var err error
		var packet *ClientMsg
		c.conn.PongHandler()
		packet, err = c.ReadJsonClientMsg()
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
func (c *ClientConn_http) DeliverRecvMsg_http() {
	defer func() {
		c.session.DisConnectCallBack()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.recvQueue:
			if !ok {
				return
			}

			// 调用回调函数
			res, err := c.session.RequestCallBack(msg, c.conn)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)
				continue
			}

			var buf []byte
			buf, err = c.WriteJsonClientMsg(res.Tag, res.Msg)
			if err != nil {
				fmt.Printf("client msg is err:%v\n", err)
				continue
			}

			c.sendQueue <- buf
		}
	}
}

// 把消息发送给客户端
func (c *ClientConn_http) WriteMsg_http() {
	defer func() {
		c.session.DisConnectCallBack()
		c.conn.Close()
	}()
	for {
		select {
		case sendMsg, ok := <-c.sendQueue:
			if !ok {
				return
			}

			// 发送给客户端
			err := c.conn.WriteMessage(websocket.TextMessage, sendMsg)
			if err != nil {
				fmt.Printf("[net write msg error:%v\n", err)
			}
		}
	}
}
