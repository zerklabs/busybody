package protocol

import "github.com/zerklabs/auburn/log"

const (
	HelloMessage    int = 0
	StandardMessage int = 1

	NoCompression      int = 0
	SnappyCompression  int = 1
	DeflateCompression int = 2
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
// \                           Content                             \
// /                                                               /
// |                                                               |
// v                                                               v
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//

func NewMessage(msgtype int, comptype int, id string) *Message {
	return &Message{
		Header:     buildMessageHeader(msgtype, comptype, id),
		raw:        make([]byte, 0),
		compressed: make([]byte, 0),
	}
}

func Decode(msg []byte) (Message, error) {

	// flag word
	version, msgtype, comptype := decodeHeaderFlags(readUint32(msg[:4]))
	timestamp := decodeTimestamp(readUint64(msg[4:12]))
	id := decodeSourceId(msg[12:20])
	bodylen := int64(readUint64(msg[20:28]))
	compbodylen := int64(readUint64(msg[28:36]))

	protocol := Message{
		Header: MessageHeader{
			version:         version,
			msgType:         msgtype,
			compressionType: comptype,
			sourceId:        id,
			timestamp:       timestamp,
			bodyLen:         bodylen,
			compBodyLen:     compbodylen,
		},
	}

	protocol.compressed = msg[36:]
	if err := protocol.decompress(); err != nil {
		log.Error(err)
	}

	return protocol, nil
}
