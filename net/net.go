package net

import (
	"github.com/mimis-s/golang_tools/net/clientConn"
	"github.com/mimis-s/golang_tools/net/service"
	"github.com/mimis-s/golang_tools/net/tcp"
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
