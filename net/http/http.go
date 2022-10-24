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

	Session clientConn.ClientSession
}

func (h *Http) SetAddr(addr, protocol string, session clientConn.ClientSession) {
	h.Addr = addr
	h.Protocol = protocol
	h.Session = session
	h.GinEngine = gin.New()
}

func (h *Http) wsHandler(c *gin.Context) {
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	clientConn := clientConn.NewClientConn_http(conn, h.Session)
	go clientConn.ReadRecvMsg_http()
	go clientConn.DeliverRecvMsg_http()
	go clientConn.WriteMsg_http()
}

func (h *Http) Listen() error {
	h.GinEngine.GET("/ws", h.wsHandler)
	return h.GinEngine.Run(h.Addr)
}
