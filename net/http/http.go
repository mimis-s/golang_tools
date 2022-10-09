package http

import (
	"net"

	"github.com/gin-gonic/gin"
	"github.com/mimis-s/golang_tools/net/clientConn"
	"golang.org/x/net/websocket"
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

func (h *Http) Listen() error {
	h.GinEngine.GET("/ws", func(handler websocket.Handler) gin.HandlerFunc {
		return func(ctx *gin.Context) {
			if ctx.IsWebsocket() {
				handler.ServeHTTP(ctx.Writer, ctx.Request)
			} else {
				_, _ = ctx.Writer.WriteString("not websocket request")
			}
		}
	}(func(conn *websocket.Conn) {
		clientConn := clientConn.NewClientConn(conn)
		go clientConn.ReadRecvMsg()
		go clientConn.DeliverRecvMsg(h.CallBack)
		go clientConn.WriteMsg()
	}))
	return h.GinEngine.Run(h.Addr)
}
