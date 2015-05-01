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
	bodyLen         int64
	compBodyLen     int64

	off int // buf offset
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

func byteEncodeUint32(v uint32) []byte {
	b := make([]byte, 4)

	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)

	return b
}

func byteEncodeUint64(v uint64) []byte {
	b := make([]byte, 8)

	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)

	return b
}

func (h *MessageHeader) encodeFlags() uint32 {
	return uint32(h.version&0xf)<<28 +
		uint32(h.msgType&0xf)<<16 +
		uint32(h.compressionType&0xf)<<8
}

func readUint32(b []byte) uint32 {
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}

func readUint64(b []byte) uint64 {
	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
}

func touint64(v int64) uint64 {
	return binary.BigEndian.Uint64(byteEncodeUint64(uint64(v)))
}

func decodeHeaderFlags(v uint32) (int, int, int) {
	version := int(v>>28) & 0xf
	msgtype := int(v>>16) & 0xf
	comptype := int(v>>8) & 0xf

	return version, msgtype, comptype
}

func decodeTimestamp(v uint64) int64 {
	return int64(v)
}

func decodeSourceId(v []byte) string {
	return string(v)
}

func (h *MessageHeader) Read(p []byte) (n int, err error) {
	bytebuf := bytes.NewBuffer(nil)

	flagsWord := h.encodeFlags()
	tsWord := touint64(h.timestamp)

	buf := byteEncodeUint32(flagsWord)
	// log.Infof("flags: %v, %d bytes", buf, len(buf))
	bytebuf.Write(buf)

	buf64 := byteEncodeUint64(tsWord)
	// log.Infof("timestamp: %v, %d bytes", buf64, len(buf64))
	bytebuf.Write(buf64)

	buf64 = byteEncodeUint64(binary.BigEndian.Uint64([]byte(h.sourceId)))
	// log.Infof("source id: %v, %d bytes", buf64, len(buf64))
	bytebuf.Write(buf64)

	buf64 = byteEncodeUint64(touint64(h.bodyLen))
	// log.Infof("raw body length: %v, %d bytes", buf64, len(buf64))
	bytebuf.Write(buf64)

	buf64 = byteEncodeUint64(touint64(h.compBodyLen))
	// log.Infof("comp body length: %v, %d bytes", buf64, len(buf64))
	bytebuf.Write(buf64)

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
