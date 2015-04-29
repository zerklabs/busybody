package protocol

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"testing"
)

func TestNewMessage(t *testing.T) {
	hash := crc32.NewIEEE()
	hash.Write([]byte("12345"))

	id := fmt.Sprintf("%x", hash.Sum(nil))
	msg := NewMessage(StandardMessage, NoCompression, id)

	buf := bytes.NewBuffer(nil)
	n, err := msg.Write(buf)
	if err != nil {
		t.Error(err)
	}

	t.Logf("wrote %d bytes", n)
	t.Logf("message %#v", buf.Bytes())
	msg.Print()
}

func TestDecode(t *testing.T) {
	hash := crc32.NewIEEE()
	hash.Write([]byte("12345"))

	id := fmt.Sprintf("%x", hash.Sum(nil))
	msg := NewMessage(StandardMessage, SnappyCompression, id)

	contentBuffer := bytes.NewBuffer(nil)
	contentBuffer.WriteString("Supercalifragilisticexpialidocious")
	msg.ReadFrom(contentBuffer)

	buf := bytes.NewBuffer(nil)
	if _, err := msg.Write(buf); err != nil {
		t.Error(err)
	}

	// t.Logf("original message: %#v", msg)

	decmsg, err := Decode(buf.Bytes())
	if err != nil {
		t.Error(err)
	}

	if decmsg.Header.msgType != msg.Header.msgType {
		t.Errorf("decoding msg type failed")
	}

	if decmsg.Header.compressionType != msg.Header.compressionType {
		t.Errorf("decoding compression type failed")
	}

	if decmsg.Header.sourceId != id {
		t.Errorf("decoding source id header failed, found %s, expected %s", decmsg.Header.sourceId, id)
	}

	if decmsg.Header.timestamp != msg.Header.timestamp {
		t.Errorf("decoding source timestamp header failed, found %d, expected %d", decmsg.Header.timestamp, msg.Header.timestamp)
	}

	if !bytes.Equal(decmsg.raw, msg.raw) {
		t.Errorf("decoding message content failed, found %s, expected %s", string(decmsg.raw), string(msg.raw))
	}

	if !bytes.Equal(decmsg.compressed, msg.compressed) {
		t.Errorf("decoding compressed message content failed, found %v, expected %v", decmsg.compressed, msg.compressed)
	}

	t.Logf("original content: %s, decoded content: %s", string(msg.raw), string(decmsg.raw))

	// decmsg.Print()
}
