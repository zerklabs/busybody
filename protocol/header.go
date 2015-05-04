package protocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"time"

	"github.com/zerklabs/auburn/log"
)

type MessageHeader struct {
	lock sync.Mutex

	version         int
	msgType         int
	compressionType int
	sourceId        string
	timestamp       int64
	bodyLen         int
	compBodyLen     int

	off int // buf offset
}

// func byteEncodeUint32(v uint32) []byte {
// 	b := make([]byte, 4)

// 	b[0] = byte(v >> 24)
// 	b[1] = byte(v >> 16)
// 	b[2] = byte(v >> 8)
// 	b[3] = byte(v)

// 	return b
// }

// func byteEncodeUint64(v uint64) []byte {
// 	b := make([]byte, 8)

// 	b[0] = byte(v >> 56)
// 	b[1] = byte(v >> 48)
// 	b[2] = byte(v >> 40)
// 	b[3] = byte(v >> 32)
// 	b[4] = byte(v >> 24)
// 	b[5] = byte(v >> 16)
// 	b[6] = byte(v >> 8)
// 	b[7] = byte(v)

// 	return b
// }

// func readUint32(b []byte) uint32 {
// 	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
// }

// func readUint64(b []byte) uint64 {
// 	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
// 		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
// }

// func touint64(v int64) uint64 {
// 	return binary.BigEndian.Uint64(byteEncodeUint64(uint64(v)))
// }

func rebuildHeader(b []byte) MessageHeader {
	buf := bytes.NewBuffer(b)
	versionb := make([]byte, 1)
	msgtypeb := make([]byte, 1)
	comptypeb := make([]byte, 1)
	timestampb := make([]byte, 8)
	sourceidb := make([]byte, 8)
	bodylenb := make([]byte, 4)
	compbodylenb := make([]byte, 4)

	binary.Read(buf, binary.LittleEndian, &versionb)
	binary.Read(buf, binary.LittleEndian, &msgtypeb)
	binary.Read(buf, binary.LittleEndian, &comptypeb)
	binary.Read(buf, binary.LittleEndian, &bodylenb)
	binary.Read(buf, binary.LittleEndian, &compbodylenb)
	binary.Read(buf, binary.BigEndian, &timestampb)
	binary.Read(buf, binary.BigEndian, &sourceidb)

	log.Infof("%v", b[:4])
	header := MessageHeader{
		version:         int(versionb[0]),
		msgType:         int(msgtypeb[0]),
		compressionType: int(comptypeb[0]),
		timestamp:       int64(binary.BigEndian.Uint64(timestampb)),
		sourceId:        string(sourceidb),
		bodyLen:         int(binary.LittleEndian.Uint32(bodylenb)),
		compBodyLen:     int(binary.LittleEndian.Uint32(compbodylenb)),
		off:             0,
	}

	// log.Infof("versionb: %v/%d", versionb, header.version)
	// log.Infof("msgtypeb: %v/%d", msgtypeb, header.msgType)
	// log.Infof("comptypeb: %v/%d", comptypeb, header.compressionType)
	// log.Infof("timestampb: %v/%d", timestampb, header.timestamp)
	// log.Infof("bodylenb: %v/%d", bodylenb, header.bodyLen)
	// log.Infof("compbodylenb: %v/%d", compbodylenb, header.compBodyLen)
	// log.Infof("sourceidb: %v/%s", sourceidb, header.sourceId)

	return header
}

func buildMessageHeader(msgtype int, comptype int, id string) MessageHeader {
	now := time.Now().UnixNano()

	return MessageHeader{
		version:         1,
		msgType:         msgtype,
		sourceId:        id,
		timestamp:       now,
		compressionType: comptype,
		bodyLen:         0,
		compBodyLen:     0,
		off:             0, // offset
	}
}

func (h *MessageHeader) Print() {
	log.Infof("%#v", h)
}

func (h *MessageHeader) Read(p []byte) (n int, err error) {
	bytebuf := bytes.NewBuffer(nil)

	binary.Write(bytebuf, binary.LittleEndian, uint8(h.version))
	binary.Write(bytebuf, binary.LittleEndian, uint8(h.msgType))
	binary.Write(bytebuf, binary.LittleEndian, uint8(h.compressionType))
	binary.Write(bytebuf, binary.LittleEndian, uint32(h.bodyLen))
	binary.Write(bytebuf, binary.LittleEndian, uint32(h.compBodyLen))
	binary.Write(bytebuf, binary.BigEndian, uint64(h.timestamp))
	binary.Write(bytebuf, binary.BigEndian, []byte(h.sourceId))

	if h.off >= bytebuf.Len() {
		if len(p) == 0 {
			return
		}
		return 0, io.EOF
	}

	n = copy(p, bytebuf.Bytes()[h.off:])
	h.off += n

	return n, nil
}
