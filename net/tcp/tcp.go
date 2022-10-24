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

	Session clientConn.ClientSession
}

func (t *Tcp) SetAddr(addr, protocol string, session clientConn.ClientSession) {
	t.Addr = addr
	t.Protocol = protocol
	t.Session = session
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
		clientConn := clientConn.NewClientConn(conn, t.Session)
		go clientConn.ReadRecvMsg()
		go clientConn.DeliverRecvMsg()
		go clientConn.WriteMsg()
	}
	return nil
}
