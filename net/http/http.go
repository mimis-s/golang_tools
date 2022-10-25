package http

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mimis-s/golang_tools/net/clientConn"
)

// type ClientSession interface {
// 	ConnectCallBack()                               // 客户端连接回调
// 	RequestCallBack(*ClientMsg) (*ClientMsg, error) // 消息处理的回调
// 	DisConnectCallBack()                            // 客户端断开连接回调
// }

type Http struct {
	Addr      string
	Protocol  string
	GinEngine *gin.Engine
	Conn      net.Conn

	NewSessionFunc func(clientConn.ClientConn) clientConn.ClientSession
}

func (h *Http) SetAddr(addr, protocol string, newSessionFunc func(clientConn.ClientConn) clientConn.ClientSession) {
	h.Addr = addr
	h.Protocol = protocol
	h.NewSessionFunc = newSessionFunc
	h.GinEngine = gin.New()
}

func (h *Http) wsHandler(c *gin.Context) {
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	clientConn := clientConn.NewClientConn_http(conn)
	session := h.NewSessionFunc(clientConn)
	go clientConn.ReadRecvMsg_http(session)
	go clientConn.DeliverRecvMsg_http(session)
	go clientConn.WriteMsg_http(session)
}

func (h *Http) Listen() error {
	h.GinEngine.GET("/ws", h.wsHandler)
	return h.GinEngine.Run(h.Addr)
}
