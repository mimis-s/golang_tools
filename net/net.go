package net

import (
	"github.com/mimis-s/golang_tools/net/clientConn"
	"github.com/mimis-s/golang_tools/net/http"
	"github.com/mimis-s/golang_tools/net/service"
	"github.com/mimis-s/golang_tools/net/tcp"
)

var mapProtol = make(map[string]service.Service)

func init() {
	mapProtol["tcp"] = new(tcp.Tcp)
	mapProtol["http"] = new(http.Http)

	// mapProtol["udp"] = new(udp.Udp)
}

func InitServer(addr string, sProtocol string, plulgInterface clientConn.ClientSession) service.Service {

	s := mapProtol[sProtocol]
	s.SetAddr(addr, sProtocol, plulgInterface)
	return s
}
