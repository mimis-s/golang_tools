package service

import "gitee.com/mimis/golang_tool/net/clientConn"

type Service interface {
	Listen() error
	SetAddr(addr, protocol string, callBack clientConn.CallBackFunc)
}
