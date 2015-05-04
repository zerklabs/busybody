package busybody

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

const DefaultSwimInterval = "1m0s"
const DefaultSwimTimeout = "30s"

type BusyConfig struct {
	Uri                     string   `toml:"uri"`
	Peers                   []string `toml:"peers"`
	SharedKey               string   `toml:"shared_key"`
	SnappyCompression       bool     `toml:"snappy_compression"`
	ZlibCompression         bool     `toml:"Zlib_compression"`
	DeflateCompression      bool     `toml:"deflate_compression"`
	DeflateCompressionLevel int      `toml:"deflate_compression_level"`
	LogLevel                int      `toml:"log_level"`
	SwimInterval            string   `toml:"swim_interval"`
	SwimTimeout             string   `toml:"swim_timeout"`
}

func ParseConfig(config []byte) (BusyConfig, error) {
	var conf BusyConfig
	if _, err := toml.Decode(string(config), &conf); err != nil {
		return BusyConfig{}, err
	}

	if conf.Uri == "" {
		return BusyConfig{}, fmt.Errorf("uri required in config")
	}

	if conf.SwimInterval == "" {
		conf.SwimInterval = DefaultSwimInterval
	}

	if conf.SwimTimeout == "" {
		conf.SwimTimeout = DefaultSwimTimeout
	}

	if conf.SnappyCompression && conf.DeflateCompression && conf.ZlibCompression {
		return BusyConfig{}, fmt.Errorf("only one of snappy, deflate or zlib can be used")
	}

	return conf, nil
}
