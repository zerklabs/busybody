package protocol

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"testing"
)

func testhostname() string {
	hash := crc32.NewIEEE()
	hash.Write([]byte("12345"))

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage(StandardMessage, NoCompression, testhostname())

	n, err := msg.Write([]byte("this is a message"))
	if err != nil {
		t.Error(err)
	}

	t.Logf("wrote %d bytes", n)
	msg.Print()
}

func TestNewMessage_Snappy(t *testing.T) {
	msg := NewMessage(StandardMessage, SnappyCompression, testhostname())

	n, err := msg.Write([]byte("this is a message"))
	if err != nil {
		t.Error(err)
	}

	t.Logf("wrote %d bytes", n)
	msg.Print()
}

func TestNewMessage_Deflate(t *testing.T) {
	msg := NewMessage(StandardMessage, DeflateCompression, testhostname())

	n, err := msg.Write([]byte("this is a message"))
	if err != nil {
		t.Error(err)
	}

	t.Logf("wrote %d bytes", n)
	msg.Print()
}

func TestDecode(t *testing.T) {
	msg := NewMessage(StandardMessage, NoCompression, testhostname())

	content := "Supercalifragilisticexpialidocious"

	msg.Write([]byte(content))

	msgb := bytes.NewBuffer(nil)
	msgb.ReadFrom(msg)

	t.Logf("msg.buf len: %d", len(msg.buf))

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

	if decmsg.Header.sourceId != testhostname() {
		t.Errorf("decoding source id header failed, found %s, expected %s", decmsg.Header.sourceId, testhostname())
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

	bb := string(body)

	if content != bb {
		t.Errorf("incorrect body decoded, found: %s, expected: %s", string(body), content)
	}
}

func TestDecode_Deflate(t *testing.T) {
	msg := NewMessage(StandardMessage, DeflateCompression, testhostname())

	content := "Supercalifragilisticexpialidocious"

	msg.Write([]byte(content))

	msgb := bytes.NewBuffer(nil)
	msgb.ReadFrom(msg)

	t.Logf("msg.buf len: %d", len(msg.buf))

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

	if decmsg.Header.sourceId != testhostname() {
		t.Errorf("decoding source id header failed, found %s, expected %s", decmsg.Header.sourceId, testhostname())
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

func TestDecode_Zlib(t *testing.T) {
	msg := NewMessage(StandardMessage, ZlibCompression, testhostname())

	content := "Supercalifragilisticexpialidocious"

	msg.Write([]byte(content))

	msgb := bytes.NewBuffer(nil)
	msgb.ReadFrom(msg)

	t.Logf("msg.buf len: %d", len(msg.buf))

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

	if decmsg.Header.sourceId != testhostname() {
		t.Errorf("decoding source id header failed, found %s, expected %s", decmsg.Header.sourceId, testhostname())
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

func TestDecode_Snappy(t *testing.T) {
	msg := NewMessage(StandardMessage, SnappyCompression, testhostname())

	content := "Supercalifragilisticexpialidocious"

	msg.Write([]byte(content))

	msgb := bytes.NewBuffer(nil)
	msgb.ReadFrom(msg)

	t.Logf("msg.buf len: %d", len(msg.buf))

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

	if decmsg.Header.sourceId != testhostname() {
		t.Errorf("decoding source id header failed, found %s, expected %s", decmsg.Header.sourceId, testhostname())
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
