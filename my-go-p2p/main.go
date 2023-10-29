package main

import (
	"context"
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
	var signalChan chan (os.Signal) = make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	appState := internal.
		NewAppState()

	appState.Config.
		WithUDPDiscoveryPort(5000).
		WithTCPAddresst(net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 6543,
		})
	log.Println("NodeId: ", appState.Config.NodeName)

	ctx := context.Background()
	ctx = context.WithValue(ctx, internal.CTX_Key_AppState, *appState)
	ctx, cancel := context.WithCancel(ctx)

	receptionModule := pkg.NewModuleReception()

	go pkg.StartBroadCast(ctx)
	go receptionModule.Start(ctx)

	go func() {
		sig := <-signalChan
		log.Printf("%s signal caught", sig)
		cancel()
	}()

	time.Sleep(1 * time.Second)
	appState.AppWaitGroup.Wait()
}
