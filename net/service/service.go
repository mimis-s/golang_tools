package service

import "golang-tool/net/clientConn"

type Service interface {
	Listen() error
	SetAddr(addr, protocol string, callBack clientConn.CallBackFunc)
}
