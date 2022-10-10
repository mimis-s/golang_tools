package http

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mimis-s/golang_tools/net/clientConn"
)

type Http struct {
	Addr      string
	Protocol  string
	GinEngine *gin.Engine
	Conn      net.Conn

	CallBack clientConn.CallBackFunc
}

func (h *Http) SetAddr(addr, protocol string, callBack clientConn.CallBackFunc) {
	h.Addr = addr
	h.Protocol = protocol
	h.CallBack = callBack
	h.GinEngine = gin.New()
}

func (h *Http) wsHandler(c *gin.Context) {
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	clientConn := clientConn.NewClientConn_http(conn)
	go clientConn.ReadRecvMsg_http()
	go clientConn.DeliverRecvMsg_http(h.CallBack)
	go clientConn.WriteMsg_http()
}

func (h *Http) Listen() error {
	h.GinEngine.GET("/ws", h.wsHandler)
	return h.GinEngine.Run(h.Addr)
}
