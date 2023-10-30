package internal

import (
	"bufio"
	"net"
	"sync"
)

type RemoteNoteConnectionState int

const (
	NONE RemoteNoteConnectionState = iota + 1
	CONNECTING
	CONNECTED
	DISCONNECTED
)

var RemoteNodeConnectionStateHumanize = []string{
	"NONE", "CONNECTING", "CONNECTED", "DISCONNECTED",
}

func (s RemoteNoteConnectionState) String() string {
	return RemoteNodeConnectionStateHumanize[s-NONE]
}

type RemoteNodeInfo struct {
	sync.Mutex

	name  string
	addr  net.TCPAddr
	conn  net.Conn
	state RemoteNoteConnectionState

	Outgoing_Chan chan ([]byte)
	Incoming_Chan chan ([]byte)

	OnStateChange func(RemoteNoteConnectionState)
}

func NewRemoteNode(name string, addr net.TCPAddr) *RemoteNodeInfo {
	return &RemoteNodeInfo{
		name:  name,
		addr:  addr,
		state: NONE,

		OnStateChange: nil,
	}
}

func (s *RemoteNodeInfo) GetState() RemoteNoteConnectionState {
	return s.state
}

func (s *RemoteNodeInfo) setState(state RemoteNoteConnectionState) {
	s.Lock()
	s.state = state
	s.Unlock()

	if s.OnStateChange != nil {
		s.OnStateChange(state)
	}
}

func (s *RemoteNodeInfo) GetName() string {
	return s.name
}

func (s *RemoteNodeInfo) Stop() error {
	err := s.conn.Close()
	if err != nil {
		return err
	}

	s.setState(DISCONNECTED)
	close(s.Incoming_Chan)
	close(s.Outgoing_Chan)

	return nil
}

func (s *RemoteNodeInfo) Start() error {
	s.setState(CONNECTING)
	conn, err := net.Dial("tcp", s.addr.String())
	if err != nil {
		s.setState(DISCONNECTED)
		return err
	}
	s.conn = conn
	s.Outgoing_Chan = make(chan []byte)
	s.Incoming_Chan = make(chan []byte)
	s.setState(CONNECTED)

	go s.startNetworkWriteStream()
	go s.startNetworkReadStream()

	return nil
}

func (s *RemoteNodeInfo) onReadStreamError() {
	// already disconnect -> ignore
	if s.GetState() == DISCONNECTED {
		return
	}
	s.Stop()
}

func (s *RemoteNodeInfo) startNetworkReadStream() {
	for {
		reader := bufio.NewReader(s.conn)
		data, err := reader.ReadBytes(0)
		if err != nil {
			s.onReadStreamError()
			return
		}

		if s.GetState() == CONNECTED {
			s.Incoming_Chan <- data
		}
	}
}

func (s *RemoteNodeInfo) startNetworkWriteStream() {
	for data := range s.Outgoing_Chan {
		n, err := s.conn.Write(data)
		if n != len(data) || err != nil {
			return
		}
	}
}
