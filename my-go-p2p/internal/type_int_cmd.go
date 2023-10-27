package internal

import "net"

type CMDType int

const (
	CMDAddNode CMDType = iota + 1
	CMDRemoveNode
)

type CMD_Any interface {
	GetType() CMDType
}

type CMD_AddNode struct {
	Addr net.Addr
}

var _ CMD_Any = (*CMD_AddNode)(nil)

func (s CMD_AddNode) GetType() CMDType {
	return CMDAddNode
}
