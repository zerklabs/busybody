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

	_, err := msg.Write([]byte("this is a message"))
	if err != nil {
		t.Error(err)
	}

	// t.Logf("wrote %d bytes", n)
	// msg.Print()
}

func TestDecode(t *testing.T) {
	hash := crc32.NewIEEE()
	hash.Write([]byte("12345"))

	id := fmt.Sprintf("%x", hash.Sum(nil))
	msg := NewMessage(StandardMessage, SnappyCompression, id)

	content := "Supercalifragilisticexpialidocious"

	msg.Write([]byte(content))

	msgb := bytes.NewBuffer(nil)
	msgb.ReadFrom(msg)

	// t.Logf("msg bytes: %v", msgb.Bytes())

	decmsg, err := Decode(msgb.Bytes())
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

	if !bytes.Equal(decmsg.buf, msg.buf) {
		t.Errorf("decoding message content failed, found:\n%v\nexpected:\n%v\n", decmsg.buf, msg.buf)
	}

	body, err := decmsg.Body()
	if err != nil {
		t.Error(err)
	}

	if content != string(body) {
		t.Errorf("incorrect body decoded")
	}
}
