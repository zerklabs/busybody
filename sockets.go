package busybody

import (
	"fmt"

	"github.com/gdamore/mangos"
	"github.com/gdamore/mangos/protocol/bus"
	"github.com/gdamore/mangos/protocol/surveyor"
	"github.com/gdamore/mangos/transport/tcp"
)

func createSurveySock(options map[string]interface{}) (mangos.Socket, error) {
	var sock mangos.Socket
	var err error

	if sock, err = surveyor.NewSocket(); err != nil {
		return nil, err
	}

	tp := tcp.NewTransport()

	sock.AddTransport(tp)

	for k, v := range options {
		if err := sock.SetOption(k, v); err != nil {
			return nil, fmt.Errorf("error setting option %s: %v", k, err)
		}
	}

	return sock, nil
}

func createBusSock(options map[string]interface{}) (mangos.Socket, error) {
	var sock mangos.Socket
	var err error

	if sock, err = bus.NewSocket(); err != nil {
		return sock, err
	}

	tp := tcp.NewTransport()
	sock.AddTransport(tp)

	for k, v := range options {
		if err := sock.SetOption(k, v); err != nil {
			return nil, fmt.Errorf("error setting option %s: %v", k, err)
		}
	}

	return sock, nil
}
