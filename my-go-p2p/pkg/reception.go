package pkg

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

type ModuleReception struct {
	ctx            context.Context
	mRWMutex       sync.RWMutex
	mRemoteNodeMap map[string]*internal.RemoteNodeInfo
}

func NewModuleReception() *ModuleReception {
	return &ModuleReception{
		mRWMutex:       sync.RWMutex{},
		mRemoteNodeMap: make(map[string]*internal.RemoteNodeInfo),
	}
}

func (s *ModuleReception) HasRemoteNodeName(nodeName string) bool {
	_, exists := s.mRemoteNodeMap[nodeName]
	return exists
}

func (s *ModuleReception) addRemoteNode(nodeInfo *internal.RemoteNodeInfo) {
	nodeName := nodeInfo.GetName()
	s.mRemoteNodeMap[nodeName] = nodeInfo

	nodeInfo.OnStateChange = func(rncs internal.RemoteNoteConnectionState) {
		if rncs == internal.DISCONNECTED {
			s.removeRemoteNode(nodeName)
			nodeInfo.OnStateChange = nil
		}
	}

	go s.startReadStreamRemoteNode(nodeInfo)

	log.Println("[ModuleReception] add node", nodeInfo.GetName(), nodeInfo.GetAddr())
}

func (s *ModuleReception) startReadStreamRemoteNode(nodeInfo *internal.RemoteNodeInfo) {
	appState, ok := s.ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}
	for data := range nodeInfo.Incoming_Chan {
		body, err := internal.Bytes2GenericMsg(data)
		if err != nil {
			log.Println(err)
			continue
		}
		appState.Chan_peer_message <- internal.CMD_PeerMessage{
			NodeName: nodeInfo.GetName(),
			Data:     body,
		}
	}
}

func (s *ModuleReception) removeRemoteNode(nodeName string) {
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	if _, ok := s.mRemoteNodeMap[nodeName]; ok {
		delete(s.mRemoteNodeMap, nodeName)
		log.Println("[ModuleReception] delete node", nodeName)
	}
}

func (s *ModuleReception) Start(ctx context.Context) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}
	log.Println("[ModuleReception]", "start")
	appState.AppWaitGroup.Add(1)
	defer func() {
		appState.AppWaitGroup.Done()
		log.Println("[ModuleReception]", "stop")
	}()

	s.ctx = ctx

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		s.runCmdLoop(ctx)
		defer wg.Done()
	}()

	go func() {
		s.runTcpListener(ctx)
		defer wg.Done()
	}()

	wg.Wait()
}

func (s *ModuleReception) runTcpListener(ctx context.Context) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}

	log.Println("[ModuleReception][runTcpListener] start")

	addr := appState.Config.Tcp_address

	listener, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	log.Println("[ModuleReception][runTcpListener] listern on ", addr)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			return
		}

		go s.handlePeerRequestConnect(conn)
	}
}

func (s *ModuleReception) runCmdLoop(ctx context.Context) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}
	log.Println("[ModuleReception][runCmdLoop] start")
	for {
		select {
		case <-ctx.Done():
			{
				log.Println("[ModuleReception][runCmdLoop] stop")
				return
			}
		case v := <-appState.Chan_reception_cmd:
			{
				switch v.GetType() {
				case internal.CmdAddNode:
					{
						cmd := v.(internal.CMD_AddNode)
						s.handleCmdAddNode(cmd)
					}
				case internal.CmdRemoveNode:
					{
						cmd := v.(internal.CMD_RemoveNode)
						s.handleCmdRemoveNode(cmd)

					}
				case internal.CmdAdminListNode:
					{
						cmd := v.(internal.CMD_CmdAdminListNode)
						s.handleCmdAdminListNode(cmd)
					}
				case internal.CmdBroadcastAllNode:
					{
						cmd := v.(internal.CMD_BroadcastAllNode)
						s.handleCmdBroadcastAllNode(cmd)
					}
				case internal.CmdSendToNode:
					{
						cmd := v.(internal.CMD_SendToNode)
						s.handleCmdSendToNode(cmd)
					}
				}
			}

		}
	}
}

func (s *ModuleReception) handleCmdSendToNode(cmd internal.CMD_SendToNode) {
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	data, err := internal.Msg2Bytes(cmd.Data)
	if err != nil {
		return
	}

	remoteNode := s.mRemoteNodeMap[cmd.NodeId]
	if remoteNode == nil {
		return
	}

	remoteNode.Outgoing_Chan <- data
}

func (s *ModuleReception) handleCmdBroadcastAllNode(cmd internal.CMD_BroadcastAllNode) {
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	data, err := internal.Msg2Bytes(cmd.Data)
	if err != nil {
		return
	}

	for _, v := range s.mRemoteNodeMap {
		v.Outgoing_Chan <- data
	}
}

func (s *ModuleReception) handleCmdAdminListNode(cmd internal.CMD_CmdAdminListNode) {
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	res := make([]internal.CMD_CmdAdminListNodeReplyItem, 0)
	for _, v := range s.mRemoteNodeMap {
		res = append(res, internal.CMD_CmdAdminListNodeReplyItem{
			Name: v.GetName(),
			Addr: v.GetAddr(),
		})
	}
	cmd.Reply <- res
}

func (s *ModuleReception) handleCmdRemoveNode(cmd internal.CMD_RemoveNode) {
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	s.removeRemoteNode(cmd.NodeName)
}

func (s *ModuleReception) handleCmdAddNode(cmd internal.CMD_AddNode) {
	appState, ok := s.ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}

	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	// ignore if node exist
	if s.HasRemoteNodeName(cmd.NodeName) {
		return
	}

	remoteNodeInfo := internal.NewRemoteNodeClient(cmd.NodeName, cmd.Addr)

	remoteNodeInfo.StartConnectTo()
	msg := internal.BuildHandshakeMessage(appState.Config.NodeName)
	data, err := internal.Msg2Bytes(msg)
	if err != nil {
		return
	}
	remoteNodeInfo.Outgoing_Chan <- data
	s.addRemoteNode(remoteNodeInfo)
}

func (s *ModuleReception) handlePeerRequestConnect(conn net.Conn) {
	data := make([]byte, 256)

	// 5 seconds for handshake
	// conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(data)
	if err != nil {
		conn.Close()
		return
	}
	msg, err := internal.Bytes2Msg[internal.HandshakeMessage](data[:n])
	if err != nil {
		conn.Close()
		return
	}

	// conn.SetDeadline(time.Time{})

	log.Println("[ModuleReception][runTcpListener] accept new connection ", conn.RemoteAddr(), msg.Body.NodeName)
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()

	remoteNodeInfo := internal.NewRemoteNodePeer(msg.Body.NodeName, conn)
	remoteNodeInfo.StartRemotePeer()
	s.addRemoteNode(remoteNodeInfo)

}
