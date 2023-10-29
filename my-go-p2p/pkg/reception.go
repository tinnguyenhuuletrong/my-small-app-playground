package pkg

import (
	"context"
	"log"
	"sync"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

type ModuleReception struct {
	mRWMutex       sync.RWMutex
	mRemoteNodeMap map[string]internal.RemoteNodeInfo
}

func NewModuleReception() *ModuleReception {
	return &ModuleReception{
		mRWMutex:       sync.RWMutex{},
		mRemoteNodeMap: make(map[string]internal.RemoteNodeInfo),
	}
}

func (s *ModuleReception) HasRemoteNodeName(nodeName string) bool {
	s.mRWMutex.RLock()
	defer s.mRWMutex.RUnlock()
	_, exists := s.mRemoteNodeMap[nodeName]
	return exists
}

func (s *ModuleReception) addRemoteNode(nodeInfo internal.RemoteNodeInfo) {
	s.mRWMutex.Lock()
	defer s.mRWMutex.Unlock()
	s.mRemoteNodeMap[nodeInfo.GetName()] = nodeInfo
	log.Println("[ModuleReception] add node", nodeInfo.GetName())
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

	var wg sync.WaitGroup
	wg.Add((1))
	go func() {
		s.runCmdLoop(ctx)
		defer wg.Done()
	}()

	wg.Wait()
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
						remoteNodeInfo := internal.NewRemoveNoteInfo(cmd.NodeName, cmd.Addr)
						s.addRemoteNode(*remoteNodeInfo)
						continue
					}
				case internal.CmdRemoveNode:
					{
						cmd := v.(internal.CMD_RemoveNode)
						s.removeRemoteNode(cmd.NodeName)
						continue
					}
				}
			}

		}
	}
}
