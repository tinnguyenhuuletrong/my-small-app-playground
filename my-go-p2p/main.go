package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/pkg"
)

func main() {
	// Args
	nodeName := flag.String("name", "node:"+internal.GetHostName()+":"+internal.GenerateRandomString(5), "nodeName")
	port := flag.Int("port", int(internal.GenRandInt(6000, 9000)), "port")
	flag.Parse()

	appState := internal.
		NewAppState()

	appState.Config.
		WithName(*nodeName).
		WithUDPDiscoveryPort(9998).
		WithTCPAddress(net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: *port,
		})
	log.Println("NodeName: ", appState.Config.NodeName)
	log.Println("TCPAddress: ", appState.Config.Tcp_address.String())

	var signalChan chan (os.Signal) = make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	receptionModule := pkg.NewModuleReception()
	broadcastModule := pkg.NewModuleBroadcast()
	ternimalModule := pkg.NewModuleTerminal()

	ctx := context.Background()
	ctx = context.WithValue(ctx, internal.CTX_Key_AppState, *appState)
	ctx, cancel := context.WithCancel(ctx)

	go broadcastModule.Start(ctx)
	go receptionModule.Start(ctx)
	go ternimalModule.Start(ctx)

	go func() {
		for v := range appState.Chan_peer_message {
			log.Println("<-", v)
		}
	}()

	go func() {
		sig := <-signalChan
		log.Printf("%s signal caught", sig)
		cancel()
	}()

	time.Sleep(1 * time.Second)
	appState.AppWaitGroup.Wait()
}
