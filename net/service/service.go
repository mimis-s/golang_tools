package service

import "github.com/mimis-s/golang_tools/net/clientConn"

type Service interface {
	Listen() error
	SetAddr(addr, protocol string, callBack clientConn.ClientSession)
}
