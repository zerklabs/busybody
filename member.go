package busybody

import (
	// "github.com/gdamore/mangos/transport/ipc"
	// "github.com/gdamore/mangos/transport/tlstcp"

	"fmt"
	"hash/crc32"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/mangos"
	"github.com/gdamore/mangos/protocol/bus"
	"github.com/gdamore/mangos/transport/tcp"
	"github.com/zerklabs/auburn/log"
)

// tls+tcp://x.x.x.x:ZZZZ

type BusyMember struct {
	msgsync sync.Mutex

	lock      sync.Mutex
	sock      mangos.Socket
	config    BusyConfig
	id        string
	hostname  string
	peers     []Introduction
	terminate bool
	handlers  []Handler

	incomingMessages chan BusyMessage
	StopChan         chan int

	introTicker     *time.Ticker
	peerShareTicker *time.Ticker
}

func initsock() (mangos.Socket, error) {
	var sock mangos.Socket
	var err error
	// var msg []byte

	if sock, err = bus.NewSocket(); err != nil {
		// log.Errorf("bus.NewSocket: %s", err)
		return sock, err
	}

	sock.AddTransport(tcp.NewTransport())

	// sock.Listen()

	return sock, nil
}

func hashHostname() string {
	crchash := crc32.NewIEEE()
	crchash.Write([]byte(hostname))

	return fmt.Sprintf("%x", crchash.Sum(nil))
}

func New(config []byte) (BusyMember, error) {
	sock, err := initsock()
	if err != nil {
		return BusyMember{}, err
	}

	if len(config) == 0 {
		return BusyMember{}, fmt.Errorf("configuration missing")
	}

	conf, err := ParseConfig(config)
	if err != nil {
		return BusyMember{}, err
	}

	introDuration, _ := time.ParseDuration(conf.IntroInterval)
	peerShareDuration, _ := time.ParseDuration(conf.PeerShareInterval)

	member := BusyMember{
		hostname:         hostname,
		id:               hashHostname(),
		sock:             sock,
		config:           conf,
		terminate:        false,
		introTicker:      time.NewTicker(introDuration),
		peerShareTicker:  time.NewTicker(peerShareDuration),
		peers:            make([]Introduction, 0),
		incomingMessages: make(chan BusyMessage, 5),
		StopChan:         make(chan int),
		handlers:         make([]Handler, 0),
	}

	for _, v := range member.config.Peers {
		if err := member.AddPeer(v); err != nil {
			return BusyMember{}, err
		}
	}

	return member, nil
}

// Generates an introduction message for this node
func (m *BusyMember) Introduction() Introduction {
	return Introduction{
		Key: m.config.SharedKey,
		Id:  m.id,
		Uri: m.config.Uri,
	}
}

func (m *BusyMember) AddPeer(peer string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// we don't want to add empty peer addresses
	if peer == "" {
		return fmt.Errorf("peer cannot be empty")
	}

	// make sure the stored uri is normalized
	peer = strings.ToLower(peer)

	exists := false

	for _, v := range m.peers {
		if v.Uri == peer {
			exists = true
		}
	}

	if !exists {
		m.peers = append(m.peers, Introduction{Uri: peer})
		if err := m.sock.Dial(peer); err != nil {
			return err
		}

		// pause for join
		time.Sleep(time.Second)
		if m.config.LogLevel >= log.INFO {
			log.Infof("successfully connected to: %s", peer)
		}
	}

	return nil
}

func (m *BusyMember) updatePeer(intro Introduction) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	exists := false

	for i, v := range m.peers {
		if v.Uri == intro.Uri {
			exists = true
			if v.Id == "" && v.Key == "" {
				m.peers[i] = intro

				if m.config.LogLevel >= log.INFO {
					log.Infof("updated peer: %#v", intro)
				}
			}
		}
	}

	if !exists {
		m.peers = append(m.peers, intro)
		if err := m.sock.Dial(intro.Uri); err != nil {
			return err
		}

		// pause for join
		time.Sleep(time.Second)
		if m.config.LogLevel >= log.INFO {
			log.Infof("added and successfully connected to: %s (%s)", intro.Id, intro.Uri)
		}
	}

	return nil
}

func (m *BusyMember) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.introTicker.Stop()
	m.peerShareTicker.Stop()
	m.terminate = true
	close(m.StopChan)
	close(m.incomingMessages)

	return nil
}

func (m *BusyMember) notificationLoop() {
	for {
		select {
		case <-m.introTicker.C:
			if m.config.LogLevel >= log.DEBUG {
				log.Debugf("intro ticker: current peers: %#v", m.peers)
			}
			if err := m.hello(); err != nil {
				log.Error(err)
			}
		case <-m.peerShareTicker.C:
			if m.config.PeerSharing {
				if m.config.LogLevel >= log.DEBUG {
					log.Debugf("peer share ticker: current peers: %#v", m.peers)
				}
				if err := m.share(); err != nil {
					log.Error(err)
				}
			}
		}
	}
}

func (m *BusyMember) handlerLoop() {
	for {
		message, ok := <-m.incomingMessages
		if !ok {
			goto exit
		}

		if message.Type == StandardMessage {
			for _, handler := range m.handlers {
				handler.HandleMessage(&message)
			}
		}

		if message.Type == HelloMessage {
			intro, err := UnmarshalIntroduction(&message)
			if err != nil {
				log.Error(err)
				continue
			}

			if intro.Key != m.config.SharedKey {
				if m.config.LogLevel >= log.WARN {
					log.Warnf("received unauthorized introduction: %#v", intro)
				}
				continue
			}

			// log.Infof("received authorized introduction: %#v", intro)
			if err := m.updatePeer(intro); err != nil {
				log.Error(err)
			}
		}
	}

exit:
	log.Infof("stopping handler")
	m.Close()
}

func (m *BusyMember) AddHandler(handler Handler) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.handlers = append(m.handlers, handler)
}

func (m *BusyMember) Listen() error {
	var msg []byte
	var err error

	defer m.Close()

	// start our listener
	if err := m.sock.Listen(m.config.Uri); err != nil {
		return err
	}

	// start dealing with incoming messages
	go m.handlerLoop()
	go m.notificationLoop()

	// now sleep to give everyone a chance to start listening
	// time.Sleep(time.Second)

	for {
		if m.terminate {
			return nil
		}

		if msg, err = m.sock.Recv(); err != nil {
			return err
		}

		bmsg, err := UnmarshalBusyMessage(msg)
		if err != nil {
			log.Error(err)
			continue
		}

		// we ignore messages from ourselves
		if bmsg.Sender != m.id {
			m.incomingMessages <- bmsg
		}
	}

	return nil
}
