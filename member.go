package busybody

import (
	// "github.com/gdamore/mangos/transport/ipc"
	// "github.com/gdamore/mangos/transport/tlstcp"

	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/mangos"
	"github.com/zerklabs/auburn/log"
	"github.com/zerklabs/busybody/protocol"
)

// tls+tcp://x.x.x.x:ZZZZ

type BusyMember struct {
	msgsync sync.Mutex
	lock    sync.RWMutex

	bussock   mangos.Socket
	config    BusyConfig
	id        string
	hostname  string
	peers     []Introduction
	terminate bool
	handlers  []Handler

	incomingMessages chan *protocol.Message
	StopChan         chan int

	introTicker     *time.Ticker
	peerShareTicker *time.Ticker
}

func New(config []byte) (BusyMember, error) {
	bussock, err := newBusSocket(make(map[string]interface{}, 0))
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
		id:               crc32hash(hostname),
		bussock:          bussock,
		config:           conf,
		terminate:        false,
		introTicker:      time.NewTicker(introDuration),
		peerShareTicker:  time.NewTicker(peerShareDuration),
		peers:            make([]Introduction, 0),
		incomingMessages: make(chan *protocol.Message),
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
		intro := Introduction{Uri: peer}
		if err := m.bussock.Dial(peer); err != nil {
			return err
		}

		// flag the connection state
		intro.connected = true
		m.peers = append(m.peers, intro)

		// pause for join
		time.Sleep(time.Second)
		if m.config.LogLevel >= log.INFO {
			log.Infof("successfully connected to: %s", peer)
		}
	}

	return nil
}

func (m *BusyMember) Members() []Introduction {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.peers
}

func (m *BusyMember) DialBus(p *Introduction) error {
	if err := m.bussock.Dial(p.Uri); err != nil {
		return err
	}

	// pause for join
	time.Sleep(time.Second)
	if m.config.LogLevel >= log.INFO {
		log.Infof("added and successfully connected to: %s (%s)", p.Id, p.Uri)
	}

	p.connected = true

	return nil
}

func (m *BusyMember) updatePeer(intro Introduction) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	exists := false
	if intro.Id == m.id {
		return fmt.Errorf("cannot add self to peer list")
	}

	for i, v := range m.peers {
		if v.Uri == intro.Uri {
			exists = true
			if v.Id == "" && v.Key == "" {
				m.peers[i] = intro
				if m.peers[i].connected {
					m.peers[i] = intro
					m.peers[i].connected = true
				} else {
					if err := m.DialBus(&m.peers[i]); err != nil {
						return err
					}
				}

				if m.config.LogLevel >= log.INFO {
					log.Infof("updated peer: %#v", intro)
				}
			}
		}
	}

	if !exists {
		if err := m.DialBus(&intro); err != nil {
			return err
		}

		m.peers = append(m.peers, intro)
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
			if err := m.hello(); err != nil {
				log.Error(err)
			}
		case <-m.peerShareTicker.C:
			if m.config.PeerSharing {
				if m.config.LogLevel >= log.DEBUG {
					log.Debugf("sharing %d peers", len(m.peers))
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

		if message.MessageType() == protocol.StandardMessage {
			for _, handler := range m.handlers {
				handler.HandleMessage(message)
			}
		}

		if message.MessageType() == protocol.HelloMessage {
			intro, err := UnmarshalIntroduction(message)
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
	if err := m.bussock.Listen(m.config.Uri); err != nil {
		return err
	}

	// start dealing with incoming messages
	go m.handlerLoop()
	go m.notificationLoop()

	for {
		if m.terminate {
			return nil
		}

		if msg, err = m.bussock.Recv(); err != nil {
			return err
		}

		bmsg, err := protocol.Decode(msg)
		if err != nil {
			log.Error(err)
			continue
		}

		// we ignore messages from ourselves
		if bmsg.Sender() != m.id {
			m.incomingMessages <- bmsg
		}
	}

	return nil
}
