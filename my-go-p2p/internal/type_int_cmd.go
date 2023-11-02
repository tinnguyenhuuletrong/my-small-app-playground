package internal

import "net"

type CMDType int

const (
	CmdAddNode CMDType = iota + 1
	CmdRemoveNode

	CmdBroadcastAllNode
	CmdSendToNode

	// P2P message
	CmdPeerMessage

	// Admin cmd
	CmdAdminListNode
)

type CMD_Any interface {
	GetType() CMDType
}

type CMD_AddNode struct {
	NodeName string
	Addr     net.TCPAddr
}

var _ CMD_Any = (*CMD_AddNode)(nil)

func (s CMD_AddNode) GetType() CMDType {
	return CmdAddNode
}

type CMD_RemoveNode struct {
	NodeName string
}

var _ CMD_Any = (*CMD_RemoveNode)(nil)

func (s CMD_RemoveNode) GetType() CMDType {
	return CmdRemoveNode
}

type CMD_SendToNode struct {
	NodeId string
	Data   NetworkMessage
}

var _ CMD_Any = (*CMD_SendToNode)(nil)

func (s CMD_SendToNode) GetType() CMDType {
	return CmdSendToNode
}

type CMD_BroadcastAllNode struct {
	Data NetworkMessage
}

var _ CMD_Any = (*CMD_BroadcastAllNode)(nil)

func (s CMD_BroadcastAllNode) GetType() CMDType {
	return CmdBroadcastAllNode
}

type CMD_CmdAdminListNodeReplyItem struct {
	Name string
	Addr string
}
type CMD_CmdAdminListNode struct {
	Reply chan []CMD_CmdAdminListNodeReplyItem
}

var _ CMD_Any = (*CMD_CmdAdminListNode)(nil)

func (s CMD_CmdAdminListNode) GetType() CMDType {
	return CmdAdminListNode
}

type CMD_PeerMessage struct {
	NodeName string
	Data     map[string]any
}

var _ CMD_Any = (*CMD_PeerMessage)(nil)

func (s CMD_PeerMessage) GetType() CMDType {
	return CmdPeerMessage
}
