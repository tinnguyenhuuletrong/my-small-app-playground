package internal

import "net"

type CMDType int

const (
	CmdAddNode CMDType = iota + 1
	CmdRemoveNode

	CmdBroadcastAllNode
	CmdSendToNode
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
