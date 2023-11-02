package internal

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type RemoteNodeType uint

const (
	NODE_TYPE_CLIENT RemoteNodeType = iota + 1
	NODE_TYPE_REMOTE_PEER
)

var RemoteNodeTypeHumanize = []string{
	"NODE_TYPE_CLIENT", "NODE_TYPE_REMOTE_PEER",
}

func (s RemoteNodeType) String() string {
	return RemoteNodeTypeHumanize[s-NODE_TYPE_CLIENT]
}

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

	mType RemoteNodeType
	name  string
	addr  net.TCPAddr
	conn  net.Conn
	state RemoteNoteConnectionState

	Outgoing_Chan chan ([]byte)
	Incoming_Chan chan ([]byte)

	OnStateChange func(RemoteNoteConnectionState)
}

func NewRemoteNodeClient(name string, addr net.TCPAddr) *RemoteNodeInfo {
	return &RemoteNodeInfo{
		mType: NODE_TYPE_CLIENT,
		name:  name,
		addr:  addr,
		state: NONE,

		OnStateChange: nil,
	}
}

func NewRemoteNodePeer(name string, conn net.Conn) *RemoteNodeInfo {
	tmp := strings.Split(conn.RemoteAddr().String(), ":")
	portNum, _ := strconv.ParseInt(tmp[1], 10, 32)

	return &RemoteNodeInfo{
		mType: NODE_TYPE_REMOTE_PEER,
		name:  name,
		conn:  conn,
		addr: net.TCPAddr{
			IP:   net.ParseIP(tmp[0]),
			Port: int(portNum),
		},
		state: NONE,

		OnStateChange: nil,
	}
}

func (s *RemoteNodeInfo) GetType() RemoteNodeType {
	return s.mType
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

func (s *RemoteNodeInfo) GetAddr() string {
	return s.addr.String()
}

func (s *RemoteNodeInfo) Stop() error {
	// already disconnect -> ignore
	if s.GetState() == DISCONNECTED {
		return nil
	}

	err := s.conn.Close()
	if err != nil {
		return err
	}

	s.setState(DISCONNECTED)
	close(s.Incoming_Chan)
	close(s.Outgoing_Chan)

	return nil
}

func (s *RemoteNodeInfo) StartConnectTo() error {
	if s.mType != NODE_TYPE_CLIENT {
		return fmt.Errorf("required Node type = NODE_TYPE_CLIENT")
	}
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

func (s *RemoteNodeInfo) StartRemotePeer() error {
	if s.mType != NODE_TYPE_REMOTE_PEER {
		return fmt.Errorf("required Node type = NODE_TYPE_REMOTE_PEER")
	}

	s.Outgoing_Chan = make(chan []byte)
	s.Incoming_Chan = make(chan []byte)
	s.setState(CONNECTED)

	go s.startNetworkWriteStream()
	go s.startNetworkReadStream()

	return nil
}

func (s *RemoteNodeInfo) onReadStreamError() {
	s.Stop()
}

func (s *RemoteNodeInfo) startNetworkReadStream() {
	for {
		reader := bufio.NewReader(s.conn)
		data, err := reader.ReadBytes(DELIM_BYTE)
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
			log.Fatalln(err)
			return
		}
	}
}
