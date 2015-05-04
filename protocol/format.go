package protocol

import "fmt"

const (
	HelloMessage int = iota
	PingMessage
	PingReqMessage
	PingReplyMessage
	PingRelayMessage
	StandardMessage
)

const (
	NoCompression int = iota
	SnappyCompression
	DeflateCompression
	ZlibCompression
)

//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      Version       |      Msg Type     |    Compression Type  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Timestamp                             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Sender ID                             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Raw Length                            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                      Compressed Length                        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// /                                                               /
// \                                                               \
// /                           Content                             /
// |                                                               |
// v                                                               v
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//

func NewMessage(msgtype int, comptype int, id string) *Message {
	return &Message{
		Header: buildMessageHeader(msgtype, comptype, id),
		buf:    make([]byte, 36), // 36 bytes to cover the header copy
		off:    0,
	}
}

func Decode(msg []byte) (*Message, error) {
	if len(msg) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	// flag word
	if len(msg) < 36 {
		return nil, fmt.Errorf("header missing from message")
	}

	version, msgtype, comptype := decodeHeaderFlags(readUint32(msg[:4]))
	timestamp := decodeTimestamp(readUint64(msg[4:12]))
	id := decodeSourceId(msg[12:20])
	bodylen := int64(readUint64(msg[20:28]))
	compbodylen := int64(readUint64(msg[28:36]))

	protocol := &Message{
		Header: MessageHeader{
			version:         version,
			msgType:         msgtype,
			compressionType: comptype,
			sourceId:        id,
			timestamp:       timestamp,
			bodyLen:         bodylen,
			compBodyLen:     compbodylen,
			off:             0,
		},
		buf: make([]byte, 0),
		off: 0,
	}

	if len(msg) > 36 {
		_, err := protocol.Write(msg[36:])
		if err != nil {
			return nil, err
		}
	}

	return protocol, nil
}
