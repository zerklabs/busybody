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
	lock sync.Mutex

	Header MessageHeader

	raw        []byte
	compressed []byte
}

func (p *Message) Print() {
	log.Infof("%#v", p)
}

func (p *Message) ReadFrom(r io.Reader) (int64, error) {
	p.lock.Lock()
	defer func() {
		p.lock.Unlock()
		p.compress()
	}()

	buf := bytes.NewBuffer(nil)
	n, err := buf.ReadFrom(r)
	p.raw = buf.Bytes()

	return n, err
}

func (p *Message) Write(w io.Writer) (int64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var total int64
	var err error

	total, err = p.Header.Write(w)
	if err != nil {
		return total, err
	}

	buf := bytes.NewBuffer(p.compressed)
	n, err := buf.WriteTo(w)
	if err != nil {
		return total, err
	}
	buf.Reset()

	// final tally
	total = total + n

	return total, nil

}

func (p *Message) updateLength() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Header.bodyLen = int64(len(p.raw))
	p.Header.compBodyLen = int64(len(p.compressed))
}

func (p *Message) compress() error {
	p.lock.Lock()
	defer func() {
		p.lock.Unlock()
		p.updateLength()
	}()

	switch p.Header.compressionType {
	case NoCompression:
		p.compressed = p.raw
	case DeflateCompression:
		buf := bytes.NewBuffer(nil)
		gz, err := zlib.NewWriterLevel(buf, flate.BestCompression)
		if err != nil {
			return err
		}

		rawbuf := bytes.NewBuffer(p.raw)
		if _, err := rawbuf.WriteTo(gz); err != nil {
			return fmt.Errorf("error writing to zlib stream: %v", err)
		}

		if err := gz.Flush(); err != nil {
			return fmt.Errorf("error flushing zlib stream: %v", err)
		}

		if err := gz.Close(); err != nil {
			return fmt.Errorf("error closing zlib stream: %v", err)
		}

		rawbuf.Reset()
		p.compressed = buf.Bytes()

	case SnappyCompression:
		buf := bytes.NewBuffer(nil)
		w := snappystream.NewWriter(buf)

		rawbuf := bytes.NewBuffer(p.raw)
		if _, err := rawbuf.WriteTo(w); err != nil {
			return fmt.Errorf("error writing to snappystream: %v", err)
		}

		p.compressed = buf.Bytes()
	}

	return nil
}

func (p *Message) decompress() error {
	p.lock.Lock()
	defer func() {
		p.lock.Unlock()
		p.updateLength()
	}()

	switch p.Header.compressionType {
	case NoCompression:
		p.raw = p.compressed
	case DeflateCompression:
		buf := bytes.NewBuffer(p.compressed)
		gz, err := zlib.NewReader(buf)
		if err != nil {
			return err
		}

		rawbuf := bytes.NewBuffer(nil)
		if _, err := rawbuf.ReadFrom(gz); err != nil {
			return fmt.Errorf("error reading from zlib stream: %v", err)
		}

		if err := gz.Close(); err != nil {
			return fmt.Errorf("error closing zlib stream: %v", err)
		}

		p.raw = rawbuf.Bytes()
	case SnappyCompression:
		buf := bytes.NewBuffer(p.compressed)
		w := snappystream.NewReader(buf, false)

		rawbuf := bytes.NewBuffer(nil)
		if _, err := rawbuf.ReadFrom(w); err != nil {
			return fmt.Errorf("error reading from snappystream: %v", err)
		}

		p.raw = rawbuf.Bytes()
	}

	return nil
}
