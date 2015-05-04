package busybody

import "testing"

var testConfig = `
uri = "tcp://192.168.1.2:48888"
peers = [ "tcp://192.168.1.3:48888", "tcp://192.168.1.4:48888" ]
shared_key = "default_shared_key"
swim_interval = "1m0s"
swim_timeout = "30s"
snappy_compression = true
zlib_compression = false
deflate_compression = false
deflate_compression_level = 6
log_level = 6
`

func TestNew(t *testing.T) {
	member, err := New([]byte(testConfig))
	if err != nil {
		t.Error(err)
	}

	t.Logf("%#v", member)
}

func TestRandomPeer(t *testing.T) {
	member, err := New([]byte(testConfig))
	if err != nil {
		t.Error(err)
	}

	peer := member.randomPeer()
	if peer == nil {
		t.Errorf("expected a peer to be returned")
	}

	t.Logf("%#v", peer)
}
