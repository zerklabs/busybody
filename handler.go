package busybody

type Handler interface {
	HandleMessage(message *BusyMessage) error
}

type HandlerFunc func(message *BusyMessage) error

func (h HandlerFunc) HandleMessage(m *BusyMessage) error {
	return h(m)
}
