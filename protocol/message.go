package protocol

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"fmt"
	"io"
	"sync"

	"github.com/mreiferson/go-snappystream"
	"github.com/zerklabs/auburn/log"
)

type Message struct {
	lock   sync.Mutex
	Header MessageHeader
	buf    []byte
	off    int // read offset
}

func (m *Message) Print() {
	log.Infof("%#v", m)
}

// Version returns the message format version
func (m *Message) Version() int {
	return m.Header.version
}

// MessageType returns the message type
func (m *Message) MessageType() int {
	return m.Header.msgType
}

// CompressionType returns the type of compression used on the
// body of the message
func (m *Message) CompressionType() int {
	return m.Header.compressionType
}

// Timestamp returns the unix timestamp when the message was created
// on the sender side. (nanosecond format)
func (m *Message) Timestamp() int64 {
	return m.Header.timestamp
}

// Sender returns the string ID of the host who sent the message
func (m *Message) Sender() string {
	return m.Header.sourceId
}

func (m *Message) Body() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.ReadFrom(m)

	return buf.Bytes()[36:], nil
}

func (m *Message) Read(p []byte) (n int, err error) {
	// initial buffer to contain the header
	pbuf := make([]byte, 36)

	n, err = m.Header.Read(pbuf)
	if err != nil {
		return 0, err
	}

	switch m.Header.compressionType {
	case NoCompression:
		pbuf = append(pbuf, m.buf...)
	case DeflateCompression:
		buf := bytes.NewBuffer(m.buf)
		gz, nerr := zlib.NewReader(buf)
		if nerr != nil {
			err = nerr
			return
		}

		rawbuf := bytes.NewBuffer(nil)
		if _, nerr := rawbuf.ReadFrom(gz); nerr != nil {
			err = fmt.Errorf("error reading from zlib stream: %v", nerr)
			return
		}

		if nerr := gz.Close(); nerr != nil {
			err = fmt.Errorf("error closing zlib stream: %v", nerr)
			return
		}

		pbuf = append(pbuf, rawbuf.Bytes()...)

		// free-up resources
		rawbuf.Reset()
		buf.Reset()

	case SnappyCompression:
		buf := bytes.NewBuffer(m.buf)
		w := snappystream.NewReader(buf, false)

		rawbuf := bytes.NewBuffer(nil)
		if _, nerr := rawbuf.ReadFrom(w); nerr != nil {
			err = fmt.Errorf("error reading from snappystream: %v", nerr)
			return
		}

		pbuf = append(pbuf, rawbuf.Bytes()...)

		// free-up resources
		rawbuf.Reset()
		buf.Reset()
	}

	if m.off >= len(m.buf) {
		if len(p) == 0 {
			return
		}
		return 0, io.EOF
	}

	n = copy(p, pbuf[m.off:])
	m.off += n

	return
}

func makeSlice(n int) []byte {
	return make([]byte, n)
}

func (m *Message) Write(p []byte) (int64, error) {
	var written int64

	m.lock.Lock()
	defer m.lock.Unlock()

	m.Header.bodyLen = int64(len(p))

	switch m.Header.compressionType {
	case NoCompression:
		if len(m.buf) < len(p) {
			m.buf = makeSlice(len(p))
		}

		n := copy(m.buf, p)
		written += int64(n)
	case DeflateCompression:
		buf := bytes.NewBuffer(nil)
		gz, err := zlib.NewWriterLevel(buf, flate.BestCompression)
		if err != nil {
			return 0, err
		}

		rawbuf := bytes.NewBuffer(p)
		nn, err := rawbuf.WriteTo(gz)
		written += nn
		if err != nil {
			return written, fmt.Errorf("error writing to zlib stream: %v", err)
		}

		if err := gz.Flush(); err != nil {
			return written, fmt.Errorf("error flushing zlib stream: %v", err)
		}

		if err := gz.Close(); err != nil {
			return written, fmt.Errorf("error closing zlib stream: %v", err)
		}

		// expand the buffer to the compressed data length
		if len(m.buf) < buf.Len() {
			m.buf = makeSlice(buf.Len())
		}

		// store the compressed input
		n := copy(m.buf, buf.Bytes())
		written += int64(n)

		// free-up resources
		rawbuf.Reset()
		buf.Reset()

	case SnappyCompression:
		buf := bytes.NewBuffer(nil)
		w := snappystream.NewWriter(buf)
		rawbuf := bytes.NewBuffer(p)

		nn, err := rawbuf.WriteTo(w)
		written += nn
		if err != nil {
			return written, fmt.Errorf("error writing to snappystream: %v", err)
		}

		// expand the buffer to the compressed data length
		if len(m.buf) < buf.Len() {
			m.buf = makeSlice(buf.Len())
		}

		n := copy(m.buf, buf.Bytes())

		written += int64(n)

		// free-up resources
		rawbuf.Reset()
		buf.Reset()
	}

	m.Header.compBodyLen = int64(len(m.buf))

	return written, nil
}
