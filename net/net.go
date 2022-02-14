package net

import (
	"gitee.com/mimis/golang_tool/net/clientConn"
	"gitee.com/mimis/golang_tool/net/service"
	"gitee.com/mimis/golang_tool/net/tcp"
)

var mapProtol = make(map[string]service.Service)

func init() {
	mapProtol["tcp"] = new(tcp.Tcp)
	// mapProtol["udp"] = new(udp.Udp)
}

func InitServer(addr string, sProtocol string, plulgFunc clientConn.CallBackFunc) service.Service {

	s := mapProtol[sProtocol]
	s.SetAddr(addr, sProtocol, plulgFunc)
	return s
}
