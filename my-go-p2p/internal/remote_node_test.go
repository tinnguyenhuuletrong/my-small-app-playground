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

func Test_WithEcho(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ip := "127.0.0.1"
	port := 4301

	wg := sync.WaitGroup{}
	wg.Add(2)

	onNewClient := func(conn net.Conn) {

		peerNode := internal.NewRemoteNodePeer("peer-test-1", conn)
		err := peerNode.StartRemotePeer()
		log.Printf("server peer created")
		if err != nil {
			log.Panicln(err)
		}

		buf := <-peerNode.Incoming_Chan
		log.Printf("server recv: %s", string(buf))

		echoMsg := fmt.Sprintf("Echo %s", string(buf))
		log.Printf("server repl: %s", echoMsg)
		peerNode.Outgoing_Chan <- []byte(echoMsg)

		defer wg.Done()
	}

	go dummyTcpServer(ctx, fmt.Sprintf("%s:%d", ip, port), onNewClient)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)

		remoteNode := internal.NewRemoteNodeClient("test-1", net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		})
		remoteNode.OnStateChange = func(state internal.RemoteNoteConnectionState) {
			log.Println("state changed", state)
		}

		err := remoteNode.StartConnectTo()
		if err != nil {
			log.Panicln(err)
		}

		remoteNode.Outgoing_Chan <- append([]byte("Ping"), 0)
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

func Test_WithRemoteClose(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ip := "127.0.0.1"
	port := 4301

	wg := sync.WaitGroup{}
	wg.Add(2)

	onNewClient := func(conn net.Conn) {
		peerNode := internal.NewRemoteNodePeer("peer-test-1", conn)
		peerNode.OnStateChange = func(state internal.RemoteNoteConnectionState) {
			log.Println("server peer state changed", state)
		}

		err := peerNode.StartRemotePeer()
		log.Printf("server peer created")
		if err != nil {
			log.Panicln(err)
		}

		peerNode.Stop()
		defer wg.Done()
	}

	go dummyTcpServer(ctx, fmt.Sprintf("%s:%d", ip, port), onNewClient)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)

		remoteNode := internal.NewRemoteNodeClient("test-1", net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		})
		remoteNode.OnStateChange = func(state internal.RemoteNoteConnectionState) {
			log.Println("state changed", state)
		}

		err := remoteNode.StartConnectTo()
		if err != nil {
			log.Panicln(err)
		}

		remoteNode.Outgoing_Chan <- []byte("Ping")
		for v := range remoteNode.Incoming_Chan {
			log.Printf("client recv: %s", string(v))
			break
		}

		// disconnect
		err = remoteNode.Stop()
		if err != nil {
			log.Panicln(err)
		}
	}()

	wg.Wait()
	cancel()
}
