package net

import (
	"fmt"
	"testing"

	"github.com/mimis-s/golang_tools/net/clientConn"
)

type TcpSession struct {
}

func NewTcpSession(clientConn.ClientConn) clientConn.ClientSession {
	return &TcpSession{}
}

func (s *TcpSession) GetClientConn() clientConn.ClientConn {
	return nil
}

func (s *TcpSession) ConnectCallBack() {

}

func (s *TcpSession) RequestCallBack(reqClient *clientConn.ClientMsg) (*clientConn.ClientMsg, error) {
	fmt.Printf("client send tag:%v message:%s\n", reqClient.Tag, reqClient.Msg)
	return &clientConn.ClientMsg{
		Tag: 1,
		Msg: []byte("成功返回"),
	}, nil
}

func (s *TcpSession) DisConnectCallBack() {

}

func TestNet(t *testing.T) {
	s := InitServer("localhost:8888", "tcp", NewTcpSession)
	s.Listen()
}
