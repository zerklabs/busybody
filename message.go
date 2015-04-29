package busybody

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"time"

	"github.com/mreiferson/go-snappystream"
	"github.com/zerklabs/auburn/log"
	"github.com/zerklabs/busybody/protocol"

	"compress/zlib"
)

type Introduction struct {
	Key       string
	Id        string
	Uri       string
	connected bool
}

type BusyMessage struct {
	Body              []byte
	Received          int64
	RecipientTimezone string
	RTT               time.Duration
	Sender            string
	SenderTimezone    string
	Sent              int64
	Type              protocol.MsgType
}

func (m *BusyMember) defaultMessage() BusyMessage {
	return BusyMessage{
		Sent:           time.Now().UnixNano(),
		Received:       0,
		Sender:         m.id,
		SenderTimezone: time.Local.String(),
		Type:           protocol.StandardMessage,
	}
}

func UnmarshalIntroduction(msg *BusyMessage) (Introduction, error) {
	var intro Introduction

	if msg.Type != protocol.HelloMessage {
		return Introduction{}, fmt.Errorf("not a hello message")
	}

	buffer := bytes.NewBuffer(msg.Body)
	decoder := gob.NewDecoder(buffer)

	if err := decoder.Decode(&intro); err != nil {
		if err != io.EOF {
			return Introduction{}, fmt.Errorf("error gob decoding introduction: %v", err)
		} else {
			log.Warnf("error during gob decoding: %v", err)
		}
	}

	if intro.Id == "" {
		return Introduction{}, fmt.Errorf("invalid introduction message: id missing")
	}

	if intro.Uri == "" {
		return Introduction{}, fmt.Errorf("invalid introduction message: uri missing")
	}

	if intro.Key == "" {
		log.Warnf("shared key is empty")
	}

	return intro, nil
}

func UnmarshalBusyMessage(raw []byte, ct protocol.CompressionType) (BusyMessage, error) {
	var msg BusyMessage
	var err error
	var buffer *bytes.Buffer

	rawBuffer := bytes.NewBuffer(raw)

	if ct == protocol.SnappyCompression {
		buffer, err = unsnapBusyMessage(rawBuffer)
		if err != nil {
			return BusyMessage{}, err
		}
	}

	if ct == protocol.DeflateCompression {
		buffer, err = deflateBusyMessage(rawBuffer)
		if err != nil {
			return BusyMessage{}, err
		}
	}

	if ct == protocol.NoCompression {
		buffer = rawBuffer
	}

	if buffer == nil {
		return BusyMessage{}, fmt.Errorf("failed to decompress message")
	}

	// we send the actual payload as a gob, to avoid having to support textual encodings (json, msgpack, toml, etc..)
	decoder := gob.NewDecoder(buffer)

	if err := decoder.Decode(&msg); err != nil {
		if err != io.EOF {
			return BusyMessage{}, fmt.Errorf("error gob decoding message: %v", err)
		} else {
			log.Warnf("error during gob decoding: %v", err)
		}
	}

	msg.RecipientTimezone = time.Local.String()
	msg.Received = time.Now().UnixNano()

	// store round trip time
	sent := time.Unix(0, msg.Sent)
	recv := time.Unix(0, msg.Received)
	msg.RTT = recv.Sub(sent)

	return msg, nil
}

func deflateBusyMessage(buf *bytes.Buffer) (*bytes.Buffer, error) {
	deflatebuf := bytes.NewBuffer(nil)

	dec, err := zlib.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("error opening zlib stream: %v", err)
	}

	if _, err := deflatebuf.ReadFrom(dec); err != nil {
		log.Errorf("error reading from zlib stream: %v", err)
		return nil, err
	}

	if err := dec.Close(); err != nil {
		log.Errorf("error closing zlib stream: %v", err)
		return nil, err
	}

	return deflatebuf, nil
}

func unsnapBusyMessage(buf *bytes.Buffer) (*bytes.Buffer, error) {
	snapbuf := bytes.NewBuffer(nil)

	r := snappystream.NewReader(buf, false)

	if _, err := snapbuf.ReadFrom(r); err != nil {
		log.Errorf("error reading from snappystream: %v", err)
		return nil, err
	}

	return snapbuf, nil
}

// Send this nodes peer list to all of it's peers
func (m *BusyMember) share() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, peer := range m.peers {
		buffer := bytes.NewBuffer(nil)
		encoder := gob.NewEncoder(buffer)
		msg := m.defaultMessage()
		msg.Type = protocol.HelloMessage

		if err := encoder.Encode(peer); err != nil {
			return fmt.Errorf("error gob encoding message: %v", err)
		}

		msg.Body = buffer.Bytes()

		if err := m.send(msg); err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (m *BusyMember) hello() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	buffer := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buffer)

	msg := m.defaultMessage()
	msg.Type = protocol.HelloMessage

	if err := encoder.Encode(m.Introduction()); err != nil {
		return fmt.Errorf("error gob encoding message: %v", err)
	}

	msg.Body = buffer.Bytes()

	return m.send(msg)
}

func (m *BusyMember) Send(content []byte) error {
	// msg := m.defaultMessage()
	// msg.Body = content
	proto := protocol.NewMessage(protocol.StandardMessage, protocol.SnappyCompression, m.id)

	proto.Print()

	return nil
	// return m.send(msg)
}

func (m *BusyMember) send(msg BusyMessage) error {
	m.msgsync.Lock()
	defer m.msgsync.Unlock()

	sendbuf := bytes.NewBuffer(nil)

	encoder := gob.NewEncoder(sendbuf)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("error gob encoding message: %v", err)
	}

	if m.config.LogLevel >= log.DEBUG && msg.Type == protocol.StandardMessage {
		log.Debugf("compressing %d bytes", sendbuf.Len())
	}

	// if m.config.SnappyCompression {
	// 	return m.sendsnappy(sendbuf, msg.Type)
	// }

	// if m.config.DeflateCompression {
	// 	return m.sendflate(sendbuf, msg.Type)
	// }

	if err := m.bussock.Send(sendbuf.Bytes()); err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	return nil
}

// func (m *BusyMember) sendsnappy(buf *bytes.Buffer, mt protocol.MsgType) error {
// 	snapbuf := bytes.NewBuffer(nil)

// 	w := snappystream.NewWriter(snapbuf)

// 	if _, err := buf.WriteTo(w); err != nil {
// 		return fmt.Errorf("error writing to snappystream: %v", err)
// 	}

// 	if m.config.LogLevel >= log.DEBUG && mt == protocol.StandardMessage {
// 		log.Debugf("sending %d compressed bytes across the wire", snapbuf.Len())
// 	}

// 	if err := m.bussock.Send(snapbuf.Bytes()); err != nil {
// 		return fmt.Errorf("error sending message: %v", err)
// 	}

// 	return nil
// }

// func (m *BusyMember) sendflate(buf *bytes.Buffer, mt protocol.MsgType) error {
// 	flatebuf := bytes.NewBuffer(nil)

// 	gz, err := zlib.NewWriterLevel(flatebuf, m.config.DeflateCompressionLevel)
// 	if err != nil {
// 		return err
// 	}

// 	if _, err := buf.WriteTo(gz); err != nil {
// 		return fmt.Errorf("error writing to zlib stream: %v", err)
// 	}

// 	if err := gz.Flush(); err != nil {
// 		log.Warnf("error flushing zlib stream: %v", err)
// 	}

// 	if err := gz.Close(); err != nil {
// 		log.Warnf("error closing zlib stream: %v", err)
// 	}

// 	if m.config.LogLevel >= log.DEBUG && mt == protocol.StandardMessage {
// 		log.Debugf("sending %d compressed bytes across the wire", flatebuf.Len())
// 	}

// 	if err := m.bussock.Send(flatebuf.Bytes()); err != nil {
// 		return fmt.Errorf("error sending message: %v", err)
// 	}

// 	return nil
// }
