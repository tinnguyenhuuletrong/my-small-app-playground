package pkg

import (
	"context"
	"log"
	"time"

	"github.com/schollz/peerdiscovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

type ModuleBroadcast struct {
	ctx context.Context
}

func NewModuleBroadcast() *ModuleBroadcast {
	return &ModuleBroadcast{}
}

func (s *ModuleBroadcast) Start(ctx context.Context) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}

	// add to app level waitgroup
	appState.AppWaitGroup.Add(1)
	defer appState.AppWaitGroup.Done()

	log.Println("[ModuleBroadcast] start")

	s.ctx = ctx

	stopChan := make(chan struct{})
	go func() {
		<-ctx.Done()
		stopChan <- struct{}{}
	}()

	msg := internal.BuildUDPDiscoveryMessage(internal.UDPDiscoveryBody{
		NodeName: appState.Config.NodeName,
		TcpAddr:  appState.Config.Tcp_address,
	})
	msg_bytes, _ := internal.Msg2Bytes(msg)

	_, err := peerdiscovery.Discover(peerdiscovery.Settings{
		Payload:   msg_bytes,
		Notify:    s.onNewPeer,
		IPVersion: peerdiscovery.IPv4,
		StopChan:  stopChan,
		TimeLimit: time.Duration(-1),
		AllowSelf: true,
		Delay:     time.Second * 5,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[ModuleBroadcast] stop")
}

func (s *ModuleBroadcast) onNewPeer(d peerdiscovery.Discovered) {
	appState, ok := s.ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}

	msg, err := internal.Bytes2Msg[internal.UDPDiscoveryMessage](d.Payload)
	if err != nil {
		return
	}

	if msg.Body.NodeName == appState.Config.NodeName {
		return
	}

	appState.Chan_reception_cmd <- internal.CMD_AddNode{
		NodeName: msg.Body.NodeName,
		Addr:     msg.Body.TcpAddr,
	}
}
