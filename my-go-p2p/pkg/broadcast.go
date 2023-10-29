package pkg

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

func StartBroadCast(ctx context.Context) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}

	var wg sync.WaitGroup
	port := appState.Config.Udp_discovery_port
	msg := internal.BuildUDPDiscoveryMessage(internal.UDPDiscoveryBody{
		NodeName: appState.Config.NodeName,
		TcpAddr:  appState.Config.Tcp_address,
	})
	msg_bytes, _ := internal.Msg2Bytes(msg)

	// add to app level waitgroup
	appState.AppWaitGroup.Add(1)
	defer appState.AppWaitGroup.Done()

	go func() {
		log.Println("Broadcast Server run")
		defer wg.Done()

		// Create a UDP connection.
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		// Listen for broadcast messages.
		for {
			// Receive a broadcast message.
			buffer := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Fatal(err)
				break
			}

			discoveryPacket, err := internal.Bytes2Msg[internal.UDPDiscoveryMessage](buffer[:n])
			if err != nil {
				continue
			}

			// Print the broadcast message.
			// log.Printf("Server recieved %v", discoveryPacket)

			cmdAddNode := internal.CMD_AddNode{
				NodeName: discoveryPacket.Body.NodeName,
				Addr:     discoveryPacket.Body.TcpAddr,
			}

			appState.Chan_reception_cmd <- cmdAddNode
			break
		}
	}()

	go func() {
		time.Sleep(1 * time.Second)

		broadcastAddress := fmt.Sprintf("255.255.255.255:%d", port)

		log.Println("Broadcast Client run")
		defer wg.Done()

		// Create a UDP connection.
		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(broadcastAddress), Port: port})
		if err != nil {
			return
		}
		defer conn.Close()

		// Send the broadcast message.
		_, err = conn.Write(msg_bytes)
		if err != nil {
			return
		}

		// Close the connection.
		defer conn.Close()
	}()

	wg.Add(2)
	wg.Wait()
}
