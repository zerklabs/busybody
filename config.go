package busybody

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

const DefaultPeerShareInterval = "120s"
const DefaultIntroductionInterval = "60s"

type BusyConfig struct {
	Uri                  string   `toml:"uri"`
	Peers                []string `toml:"peers"`
	SharedKey            string   `toml:"shared_key"`
	Compression          bool     `toml:"compression"`
	WireCompression      bool     `toml:"wire_compression"`
	WireCompressionLevel int      `toml:"wire_compression_level"`
	LogLevel             int      `toml:"log_level"`
	IntroInterval        string   `toml:"introduction_interval"`
	PeerSharing          bool     `toml:"enable_peer_sharing"`
	PeerShareInterval    string   `toml:"peer_share_interval"`
}

func ParseConfig(config []byte) (BusyConfig, error) {
	var conf BusyConfig
	if _, err := toml.Decode(string(config), &conf); err != nil {
		return BusyConfig{}, err
	}

	if conf.Uri == "" {
		return BusyConfig{}, fmt.Errorf("uri required in config")
	}

	if conf.IntroInterval == "" {
		conf.IntroInterval = DefaultIntroductionInterval
	}

	if conf.PeerShareInterval == "" {
		conf.IntroInterval = DefaultPeerShareInterval
	}

	conf.WireCompression = true

	return conf, nil
}
