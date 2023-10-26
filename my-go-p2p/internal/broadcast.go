package internal

import (
	"context"
	"log"
	"net"
	"sync"
	"time"
)

func StartBroadCast(ctx context.Context) {
	var wg sync.WaitGroup

	// TODO: from config
	port := 5000

	// TODO: find my TCP port -> build message
	msg := BuildUDPDiscoveryMessage(UDPDiscoveryBody{
		Ipv4: "127.0.0.1:1234",
	})
	msg_bytes, _ := Msg2Bytes(msg)

	go func() {
		log.Println("Server run")
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
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Fatal(err)
				break
			}
			// Print the broadcast message.
			log.Printf("Server recieved %s from %s", string(buffer[:n]), addr)
			break
		}
	}()

	go func() {
		time.Sleep(2 * time.Second)

		broadcastAddress := "255.255.255.255:5000"

		log.Println("Client run")
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
