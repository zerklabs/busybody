package protocol

// func TestBuildMessageHeader(t *testing.T) {
// 	hash := crc32.NewIEEE()
// 	hash.Write([]byte("12345"))

// 	header := buildMessageHeader(StandardMessage, NoCompression, fmt.Sprintf("%x", hash.Sum(nil)))

// 	// header.Print()
// 	// mtlen := len(header.MsgTypeHdr)

// 	// if mtlen != 1 {
// 	// 	t.Errorf("expected MsgTypeHdr to be 1 byte, was %d", mtlen)
// 	// }

// 	// srcidlen := len(header.SourceIdHdr)

// 	// if srcidlen != 8 {
// 	// 	t.Errorf("expected SourceIdHdr to be 8 bytes, was %d", srcidlen)
// 	// }

// 	// srctslen := len(header.SourceTimestampHdr)

// 	// if srctslen != 19 {
// 	// 	t.Errorf("expected SourceIdHdr to be 19 bytes, was %d", srctslen)
// 	// }

// 	// ctlen := len(header.CompressionTypeHdr)

// 	// if ctlen != 1 {
// 	// 	t.Errorf("expected CompressionTypeHdr to be 1 bytes, was %d", ctlen)
// 	// }
// }
