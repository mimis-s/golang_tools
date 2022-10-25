package service

import "github.com/mimis-s/golang_tools/net/clientConn"

type Service interface {
	Listen() error
	SetAddr(addr, protocol string, newSessionFunc func(clientConn.ClientConn) clientConn.ClientSession)
}
