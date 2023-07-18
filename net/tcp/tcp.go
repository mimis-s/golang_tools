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

	NewSessionFunc func(clientConn.ClientConn) clientConn.ClientSession
}

func (t *Tcp) SetAddr(addr, protocol string, newSessionFunc func(clientConn.ClientConn) clientConn.ClientSession) {
	t.Addr = addr
	t.Protocol = protocol
	t.NewSessionFunc = newSessionFunc
}

func (t *Tcp) Run() error {
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
		session := t.NewSessionFunc(clientConn)
		go clientConn.ReadRecvMsg(session)
		go clientConn.DeliverRecvMsg(session)
		go clientConn.WriteMsg()
	}
	return nil
}
func (t *Tcp) Stop() {
	t.Listener.Close()
}
