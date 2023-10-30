package internal

import (
	"net"
	"sync"
)

type ctxKey string

const (
	CTX_Key_AppState ctxKey = "ctx_key_app_state"
)

type appConfig struct {
	NodeName           string
	Tcp_address        net.TCPAddr
	Udp_discovery_port int
}

type AppState struct {
	Config             appConfig
	AppWaitGroup       *sync.WaitGroup
	Chan_reception_cmd chan CMD_Any
}

func NewAppState() *AppState {
	return &AppState{
		Config: appConfig{
			NodeName: "node:" + GetHostName() + ":" + GenerateRandomString(5),
		},
		AppWaitGroup:       &sync.WaitGroup{},
		Chan_reception_cmd: make(chan CMD_Any),
	}
}

func (s *appConfig) WithName(name string) *appConfig {
	s.NodeName = name
	return s
}

func (s *appConfig) WithTCPAddress(tcp_address net.TCPAddr) *appConfig {
	s.Tcp_address = tcp_address
	return s
}

func (s *appConfig) WithUDPDiscoveryPort(port int) *appConfig {
	s.Udp_discovery_port = port
	return s
}
