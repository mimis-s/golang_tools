package rtmp

import (
	"bufio"
	"bytes"
	"net"
)

type Service struct {
	done          bool
	streamID      int
	isPublisher   bool
	conn          *Conn
	transactionID int
	bytesw        *bytes.Buffer
}

func (s *Service) Listen(rtmpAddr string) error {
	listen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		return err
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			return err
		}
		c := &Conn{
			conn,
			bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
			make(map[int]*Rtmp),
		}

		go s.handleConn(c)
	}

}

func (s *Service) handleConn(conn *Conn) error {
	// rtmp三次握手
	if err := conn.ShakeAsService(); err != nil {
		conn.Close()
		return err
	}

	if err := s.ReadMsg(conn); err != nil {
		conn.Close()
		return err
	}
	return nil
}

func (s *Service) ReadMsg(conn *Conn) error {
	rtmp := &Rtmp{}
	for {
		if err := conn.ReadRTMP(rtmp); err != nil {
			return err
		}
		// 协议控制消息：Message Type ID = 1~6，主要用于协议内的控制。
		// 数据消息：Message Type ID = 8 9
		// 188: Audio 音频数据
		// 9: Video 视频数据1
		// 8: Metadata 包括音视频编码、视频宽高等音视频元数据。
		// 命令消息 Command Message (20, 17)：此类型消息主要有 NetConnection 和 NetStream 两类，
		// 两类分别有多个函数，该消息的调用，可理解为远程函数调用。
		switch rtmp.MsgHeader.MsgTypeID {
		case 17, 20:

		}
	}
}

func (s *Service) HandleCmdMsg(conn *Conn, r *Rtmp) error {
	// tp := 0
	// if r.MsgHeader.MsgTypeID == 17 {
	// 	r.MsgBody.Data = r.MsgBody.Data[1:]
	// }
	return nil
}
