package net

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"gitee.com/mimis/golang-tool/lib/zbyte"
)

func sendMessage(conn net.Conn) {
	//传输map
	userMap := make(map[string]int)
	userMap["张三"] = 18

	sSendMessage, err := json.Marshal(userMap)
	if err != nil {
		log.Fatal(err)
	}

	//包id标识
	var PkgID uint32 = 12
	//要将uint32转化为字节byte[]
	PkgIDByte := make([]byte, 4)
	PkgIDByte[3] = uint8(PkgID)
	PkgIDByte[2] = uint8(PkgID >> 8)
	PkgIDByte[1] = uint8(PkgID >> 16)
	PkgIDByte[0] = uint8(PkgID >> 24)

	//包大小
	MessageLens := len(sSendMessage)
	fmt.Printf("发送的数据大小：%v\n", MessageLens)
	PkgSizeByte := make([]byte, 4)
	PkgSizeByte[3] = uint8(MessageLens)
	PkgSizeByte[2] = uint8(MessageLens >> 8)
	PkgSizeByte[1] = uint8(MessageLens >> 12)
	PkgSizeByte[0] = uint8(MessageLens >> 24)

	//组包

	TCPMessage := make([]byte, 0)
	TCPMessage = append(TCPMessage, PkgIDByte...)
	TCPMessage = append(TCPMessage, PkgSizeByte...)
	TCPMessage = append(TCPMessage, sSendMessage...)

	fmt.Printf("发送的包大小：%v\n", len(TCPMessage))

	for k := 0; k != 10; k++ {
		time.Sleep(time.Second)
		fmt.Printf("数据%s\n", TCPMessage)

		_, err = conn.Write([]byte(TCPMessage))
		if err != nil {
			log.Fatal(err)
		}

		recvHead := make([]byte, 8)
		_, err = conn.Read(recvHead)
		if err != nil {
			log.Fatal(err)
		}
		//包id标识
		recvID := zbyte.ByteToInt(recvHead[:4])
		recvLen := zbyte.ByteToInt(recvHead[4:])

		recvMsg := make([]byte, recvLen)
		_, err = conn.Read(recvMsg)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("服务器返回：tag[%v] msg:[%s]\n", recvID, recvMsg)
	}
}

func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		log.Fatal(err)
	}
	sendMessage(conn)
}
