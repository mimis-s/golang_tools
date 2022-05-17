package rtmp

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

type Conn struct {
	net.Conn
	ReadWriter *bufio.ReadWriter
	MapTrunk   map[int]*Rtmp
}

// message header
type Header struct {
	Format            int
	CSID              int
	Timestamp         int
	TimestampDelta    int
	MsgLength         int
	MsgTypeID         int
	MsgStreamID       int
	UseExtedTimestamp bool
}

// message body
type Body struct {
	ReadEnd  bool
	Index    int
	CSLength int
	Data     []byte
}

type Rtmp struct {
	MsgHeader *Header
	MsgBody   *Body
}

func (c *Conn) ReadByteBE(count int) (int, error) {
	s := make([]byte, count)
	n, err := io.ReadFull(c.ReadWriter, s)
	if err != nil {
		return 0, err
	}
	if n != len(s) {
		return 0, fmt.Errorf("read full len[%v] != %v is err", n, len(s))
	}
	sum := 0
	for i := 0; i < len(s); i++ {
		sum = sum<<(i*8) + int(s[i])
	}
	return sum, nil
}

func (c *Conn) ReadByteLE(count int) (int, error) {
	s := make([]byte, count)
	n, err := io.ReadFull(c.ReadWriter, s)
	if err != nil {
		return 0, err
	}
	if n != len(s) {
		return 0, fmt.Errorf("read full len[%v] != %v is err", n, len(s))
	}
	sum := 0
	for i := 0; i < len(s); i++ {
		sum = sum + int(s[i]<<(i*8))
	}
	return sum, nil
}

func (c *Conn) readHeader() (*Rtmp, error) {
	// basic header(1-3byte,分情况)
	fmatAndCSID, err := c.ReadByteBE(1)
	if err != nil {
		return nil, err
	}

	fmat := int(fmatAndCSID >> 6) // 左边两位为format
	csID := fmatAndCSID & 0x3f    // 0x3f十进制63，刚好表示右边六位csid,默认情况csID = 2

	switch csID {
	case 0:
		csID_2, err := c.ReadByteLE(1)
		if err != nil {
			return nil, err
		}
		csID = csID_2 + 64
	case 1:
		csID_2, err := c.ReadByteLE(2)
		if err != nil {
			return nil, err
		}
		csID = csID_2 + 64
	}

	rtmp := &Rtmp{}

	if c.MapTrunk[csID] != nil {
		rtmp = c.MapTrunk[csID]
	} else {
		rtmp.MsgHeader = &Header{
			CSID: csID,
		}
		c.MapTrunk[csID] = rtmp
	}

	header := rtmp.MsgHeader
	// message header
	switch fmat {
	case 0:
		header.Format = fmat
		var err error
		header.Timestamp, err = c.ReadByteBE(3)
		header.MsgLength, err = c.ReadByteBE(3)
		header.MsgTypeID, err = c.ReadByteBE(1)
		header.MsgStreamID, err = c.ReadByteLE(4)
		if err != nil {
			return rtmp, err
		}

		// 当3byte存不下时间戳之后，在header和body中间还有一个扩展时间戳4byte可以使用
		if header.Timestamp == 0xffffff {
			header.Timestamp, err = c.ReadByteBE(4)
			if err != nil {
				return rtmp, err
			}
			header.UseExtedTimestamp = true
		} else {
			header.UseExtedTimestamp = false
		}
	case 1:
		header.Format = fmat
		var err error
		timestampDelta, err := c.ReadByteBE(3)
		header.MsgLength, err = c.ReadByteBE(3)
		header.MsgTypeID, err = c.ReadByteBE(1)
		if err != nil {
			return rtmp, err
		}
		if timestampDelta == 0xffffff {
			timestampDelta, err = c.ReadByteBE(4)
			if err != nil {
				return rtmp, err
			}
			header.UseExtedTimestamp = true
		} else {
			header.UseExtedTimestamp = false
		}
		header.TimestampDelta = timestampDelta
		header.Timestamp += timestampDelta
	case 2:
		header.Format = fmat
		var err error
		timestampDelta, err := c.ReadByteBE(3)
		if err != nil {
			return rtmp, err
		}
		if timestampDelta == 0xffffff {
			timestampDelta, err = c.ReadByteBE(4)
			if err != nil {
				return rtmp, err
			}
			header.UseExtedTimestamp = true
		} else {
			header.UseExtedTimestamp = false
		}
		header.TimestampDelta = timestampDelta
		header.Timestamp += timestampDelta
	case 3:
		if rtmp.MsgBody.CSLength == 0 {
			switch header.Format {
			case 0:
				if header.UseExtedTimestamp {
					header.Timestamp, err = c.ReadByteBE(4)
					if err != nil {
						return nil, err
					}
				}
			case 1, 2:
				timestampDet := 0
				if header.UseExtedTimestamp {
					timestampDet, err = c.ReadByteBE(4)
					if err != nil {
						return nil, err
					}
				} else {
					timestampDet = header.TimestampDelta
				}
				header.Timestamp += timestampDet
			}
		}

	default:
		return nil, fmt.Errorf("invalid format=%d", header.Format)
	}
	return rtmp, nil
}

func (c *Conn) readBody(rtmp *Rtmp) (bool, error) {
	// body
	if rtmp.MsgBody == nil {
		rtmp.MsgBody = &Body{
			false,
			0,
			rtmp.MsgHeader.MsgLength,
			make([]byte, rtmp.MsgHeader.MsgLength),
		}
	}

	size := rtmp.MsgBody.CSLength

	if size > 128 {
		size = 128
	}

	buf := rtmp.MsgBody.Data[rtmp.MsgBody.Index : rtmp.MsgBody.Index+size]
	n, err := io.ReadFull(c.ReadWriter, buf)
	if err != nil {
		return false, err
	}
	if n != len(buf) {
		return false, fmt.Errorf("read full buf len[%v] != return len%v", len(buf), n)
	}
	rtmp.MsgBody.Index += size
	rtmp.MsgBody.CSLength -= size
	if rtmp.MsgBody.CSLength == 0 {
		return true, nil
	}
	return false, nil
}

func (c *Conn) ReadRTMP(r *Rtmp) error {
	for {
		rtmp, err := c.readHeader()
		if err != nil {
			return err
		}
		ok, err := c.readBody(rtmp)
		if err != nil {
			return err
		}
		if ok {
			r.MsgHeader = rtmp.MsgHeader
			r.MsgBody = rtmp.MsgBody
			break
		}
	}
	return nil
}
