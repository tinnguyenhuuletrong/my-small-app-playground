package internal

import "net"

// Type enum for NetworkMessages.
type NetworkMessageType uint32

type NetworkMessage interface {
	// Get the type of the message.
	GetType() NetworkMessageType

	// Get the body of the message.
	GetBody() any
}

const (
	NetworkMessageType_UDP_DISCOVERY NetworkMessageType = iota + 1
	NetworkMessageType_PING
	NetworkMessageType_PONG
)

// Implementation of NetworkMessage for UDPDiscoveryMessage message.
type UDPDiscoveryBody struct {
	NodeName string
	TcpAddr  net.TCPAddr
}
type UDPDiscoveryMessage struct {
	Body UDPDiscoveryBody
}

var _ NetworkMessage = (*UDPDiscoveryMessage)(nil)

func (m UDPDiscoveryMessage) GetType() NetworkMessageType {
	return NetworkMessageType_UDP_DISCOVERY
}

func (m UDPDiscoveryMessage) GetBody() any {
	return m.Body
}

func BuildUDPDiscoveryMessage(body UDPDiscoveryBody) *UDPDiscoveryMessage {
	return &UDPDiscoveryMessage{
		Body: body,
	}
}

// Implementation of NetworkMessage for PingMessage message.
type PingBody struct {
}
type PingMessage struct {
	Body PingBody
}

var _ NetworkMessage = (*PingMessage)(nil)

// GetBody implements NetworkMessage.
func (m PingMessage) GetBody() any {
	return m.Body
}

// GetType implements NetworkMessage.
func (m PingMessage) GetType() NetworkMessageType {
	return NetworkMessageType_PING
}

func BuildPingMessage() *PingMessage {
	return &PingMessage{
		Body: PingBody{},
	}
}

// Implementation of NetworkMessage for PongMessage message.
type PongBody struct {
}
type PongMessage struct {
	Body PongBody
}

var _ NetworkMessage = (*PongMessage)(nil)

// GetBody implements NetworkMessage.
func (m PongMessage) GetBody() any {
	return m.Body
}

// GetType implements NetworkMessage.
func (m PongMessage) GetType() NetworkMessageType {
	return NetworkMessageType_PONG
}

func BuildPongMessage() *PongMessage {
	return &PongMessage{
		Body: PongBody{},
	}
}
