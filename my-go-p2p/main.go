package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/pkg"
)

func main() {

	appState := internal.
		NewAppState()

	appState.Config.
		WithUDPDiscoveryPort(5000).
		WithTCPAddresst(net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 6543,
		})

	ctx := context.Background()
	ctx = context.WithValue(ctx, internal.CTX_Key_AppState, *appState)

	go func(ctx context.Context) {
		appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
		appState.AppWaitGroup.Add(1)
		defer appState.AppWaitGroup.Done()
		if !ok {
			log.Fatalln("ctx.appstate not exists")
			return
		}

		for itm := range appState.Chan_reception_cmd {
			switch itm.GetType() {
			case internal.CMDAddNode:
				{
					data := itm.(internal.CMD_AddNode)
					log.Println("<-", data)
				}
			}
			break
		}
	}(ctx)
	go pkg.StartBroadCast(ctx)

	time.Sleep(1 * time.Second)
	appState.AppWaitGroup.Wait()
}
