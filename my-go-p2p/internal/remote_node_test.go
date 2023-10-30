package internal_test

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

func dummyTcpServer(ctx context.Context, addr string, handleNewClient func(net.Conn)) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic("dummyTcpServer can not start", err)
		return
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		client, err := listener.Accept()
		if err != nil {
			return
		}
		go handleNewClient(client)
	}
}

func TestRemoteNodeTCP(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ip := "127.0.0.1"
	port := 4301

	wg := sync.WaitGroup{}
	wg.Add(2)

	onNewClient := func(conn net.Conn) {
		log.Println("server client connected")
		buf := make([]byte, 256)
		_, err := conn.Read(buf)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("server recv: %s", string(buf))
		echoMsg := fmt.Sprintf("Echo %s", string(buf))
		log.Printf("server repl: %s", echoMsg)
		_, err = conn.Write([]byte(echoMsg))
		if err != nil {
			log.Fatalln(err)
		}

		defer wg.Done()
	}

	go dummyTcpServer(ctx, fmt.Sprintf("%s:%d", ip, port), onNewClient)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)

		remoteNode := internal.NewRemoteNode("test-1", net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		})
		remoteNode.OnStateChange = func(state internal.RemoteNoteConnectionState) {
			log.Println("state changed", state)
		}

		err := remoteNode.Start()
		if err != nil {
			log.Panicln(err)
		}

		remoteNode.Outgoing_Chan <- []byte("Ping")
		for v := range remoteNode.Incoming_Chan {
			log.Printf("client recv: %s", string(v))
			break
		}

		// disconnect
		remoteNode.Stop()
	}()

	wg.Wait()
	cancel()
}