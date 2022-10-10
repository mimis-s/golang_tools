package tcp

import (
	"fmt"
	"net"

	"github.com/mimis-s/golang_tools/net/clientConn"
)

type Tcp struct {
	Addr     string
	Protocol string
	Listener net.Listener

	CallBack clientConn.CallBackFunc
}

func (t *Tcp) SetAddr(addr, protocol string, callBack clientConn.CallBackFunc) {
	t.Addr = addr
	t.Protocol = protocol
	t.CallBack = callBack
}

func (t *Tcp) Listen() error {
	var err error
	t.Listener, err = net.Listen(t.Protocol, t.Addr)
	if err != nil {
		return err
	}
	fmt.Printf("service start success, wait connect...\n")

	for {
		conn, err := t.Listener.Accept()
		if err != nil {
			fmt.Printf("client[%v] accept is err[%v]\n", conn.RemoteAddr().String(), err)
			continue
		}
		clientConn := clientConn.NewClientConn(conn)
		go clientConn.ReadRecvMsg()
		go clientConn.DeliverRecvMsg(t.CallBack)
		go clientConn.WriteMsg()
	}
	return nil
}
