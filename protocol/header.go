package protocol

import (
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
}

func buildMessageHeader(msgtype int, comptype int, id string) MessageHeader {
	now := time.Now().UnixNano()
	// test := uint32(msgtype&0xf)<<28 +
	// 	uint32(comptype&0xf)<<16

	// mt := int(test>>28) & 0xf
	// ct := int(test>>16) & 0xf

	// log.Infof("%d, %d, %d", test, mt, ct)

	return MessageHeader{
		version:         1,
		msgType:         msgtype,
		sourceId:        id,
		timestamp:       now,
		compressionType: comptype,
		bodyLen:         0,
		compBodyLen:     0,
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

func (h *MessageHeader) Write(w io.Writer) (int64, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	var total int64

	total = 0

	flagsWord := h.encodeFlags()
	tsWord := touint64(h.timestamp)

	log.Infof("flags: %v", byteEncodeUint32(flagsWord))
	n, err := w.Write(byteEncodeUint32(flagsWord))
	if err != nil {
		return total, err
	}

	total = total + int64(n)

	log.Infof("timestamp: %v", byteEncodeUint64(tsWord))
	n, err = w.Write(byteEncodeUint64(tsWord))
	if err != nil {
		return total, err
	}

	total = total + int64(n)

	log.Infof("source id: %v", byteEncodeUint64(binary.BigEndian.Uint64([]byte(h.sourceId))))
	n, err = w.Write(byteEncodeUint64(binary.BigEndian.Uint64([]byte(h.sourceId))))
	if err != nil {
		return total, err
	}

	total = total + int64(n)

	log.Infof("raw body length: %v", byteEncodeUint64(touint64(h.bodyLen)))
	n, err = w.Write(byteEncodeUint64(touint64(h.bodyLen)))
	if err != nil {
		return total, err
	}

	total = total + int64(n)

	log.Infof("comp body length: %v", byteEncodeUint64(touint64(h.compBodyLen)))
	n, err = w.Write(byteEncodeUint64(touint64(h.compBodyLen)))
	if err != nil {
		return total, err
	}

	total = total + int64(n)

	return total, nil
}
