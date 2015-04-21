package busybody

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/zerklabs/auburn/log"

	"code.google.com/p/snappy-go/snappy"
)

type MsgType int

const (
	HelloMessage MsgType = iota
	StandardMessage
)

type Introduction struct {
	Key string
	Id  string
	Uri string
}

type BusyMessage struct {
	Body              []byte
	Compressed        bool
	Received          int64
	RecipientTimezone string
	RTT               time.Duration
	Sender            string
	SenderTimezone    string
	Sent              int64
	Type              MsgType
}

func (m *BusyMember) defaultMessage() BusyMessage {
	return BusyMessage{
		Sent:           time.Now().UnixNano(),
		Received:       0,
		Sender:         m.id,
		Compressed:     m.config.Compression,
		SenderTimezone: time.Local.String(),
		Type:           StandardMessage,
	}
}

func UnmarshalIntroduction(msg *BusyMessage) (Introduction, error) {
	var intro Introduction

	if msg.Type != HelloMessage {
		return Introduction{}, fmt.Errorf("not a hello message")
	}

	buffer := bytes.NewBuffer(msg.Body)
	decoder := gob.NewDecoder(buffer)

	if err := decoder.Decode(&intro); err != nil {
		return Introduction{}, fmt.Errorf("error gob decoding introduction: %v", err)
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

func UnmarshalBusyMessage(raw []byte) (BusyMessage, error) {
	var msg BusyMessage

	buffer := bytes.NewBuffer(raw)

	// we send the actual payload as a gob, to avoid having to support textual encodings (json, msgpack, toml, etc..)
	decoder := gob.NewDecoder(buffer)

	if err := decoder.Decode(&msg); err != nil {
		return BusyMessage{}, fmt.Errorf("error gob decoding message: %v", err)
	}

	msg.RecipientTimezone = time.Local.String()
	msg.Received = time.Now().UnixNano()

	// store round trip time
	sent := time.Unix(0, msg.Sent)
	recv := time.Unix(0, msg.Received)
	msg.RTT = recv.Sub(sent)

	if msg.Compressed {
		rawBody := make([]byte, 0)
		b, err := snappy.Decode(rawBody, msg.Body)
		if err != nil {
			return BusyMessage{}, fmt.Errorf("error decoding snappy stream: %v", err)
		}

		// log.Infof("snappy decode: b len %d, compressed body len: %d", len(b), len(msg.Body))
		msg.Body = b
	}

	return msg, nil
}

// Send this nodes peer list to all of it's peers
func (m *BusyMember) share() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, peer := range m.peers {
		buffer := bytes.NewBuffer(nil)
		encoder := gob.NewEncoder(buffer)
		msg := m.defaultMessage()
		msg.Type = HelloMessage
		msg.Compressed = false

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
	msg.Type = HelloMessage
	msg.Compressed = false

	if err := encoder.Encode(m.Introduction()); err != nil {
		return fmt.Errorf("error gob encoding message: %v", err)
	}

	msg.Body = buffer.Bytes()

	return m.send(msg)
}

func (m *BusyMember) Send(content []byte) error {
	msg := m.defaultMessage()

	if m.config.Compression {
		compressedBody := make([]byte, 0)
		b, err := snappy.Encode(compressedBody, content)
		if err != nil {
			return fmt.Errorf("error encoding snappy stream: %v", err)
		}

		// log.Infof("snappy encode: b len %d, compressed body len: %d", len(b), len(compressedBody))
		msg.Body = b
	} else {
		msg.Body = content
	}

	return m.send(msg)
}

func (m *BusyMember) send(msg BusyMessage) error {
	m.msgsync.Lock()
	defer m.msgsync.Unlock()

	sendbuf := bytes.NewBuffer(nil)

	encoder := gob.NewEncoder(sendbuf)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("error gob encoding message: %v", err)
	}

	if m.config.LogLevel >= log.DEBUG {
		log.Debugf("sending %d bytes across the wire", sendbuf.Len())
	}

	if err := m.sock.Send(sendbuf.Bytes()); err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	return nil
}
