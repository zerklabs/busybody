package busybody

import "testing"

var testConfig = `
uri = "tcp://192.168.1.2:48888"
peers = [ "tcp://192.168.1.3:48888", "tcp://192.168.1.4:48888", "tcp://192.168.1.5:48888", "tcp://192.168.1.6:48888" ]
shared_key = "default_shared_key"
swim_interval = "1m0s"
swim_timeout = "30s"
snappy_compression = false
zlib_compression = false
deflate_compression = true
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

func TestRandomPeerGroup(t *testing.T) {
	member, err := New([]byte(testConfig))
	if err != nil {
		t.Error(err)
	}

	group := member.selectPeerGroup(&member.peers[0])
	if group == nil {
		t.Errorf("expected a non-nil slice to be returned")
	}

	if len(group) <= 0 {
		t.Errorf("expected a non-empty slice to be returned")
	}

	for i := range group {
		t.Logf("%#v", group[i])
	}
}
