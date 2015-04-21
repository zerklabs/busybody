package busybody

import (
	"os"

	"github.com/zerklabs/auburn/log"
)

var (
	hostname string
)

func init() {
	h, err := os.Hostname()
	if err != nil {
		log.Errorf("error fetching hostname: %s", err)
		os.Exit(1)
	}

	hostname = h
}
