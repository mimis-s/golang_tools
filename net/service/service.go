package service

import "gitee.com/mimis/golang-tool/net/clientConn"

type Service interface {
	Listen() error
	SetAddr(addr, protocol string, callBack clientConn.CallBackFunc)
}
