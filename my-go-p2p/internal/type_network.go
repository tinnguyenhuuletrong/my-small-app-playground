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
	NetworkMessageType_HANDSHAKE
	NetworkMessageType_STRINGDATA
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

// Implementation of NetworkMessage for HandshakeMessage message.
type HandshakeBody struct {
	NodeName string
}
type HandshakeMessage struct {
	Body HandshakeBody
}

var _ NetworkMessage = (*HandshakeMessage)(nil)

// GetBody implements NetworkMessage.
func (m HandshakeMessage) GetBody() any {
	return m.Body
}

// GetType implements NetworkMessage.
func (m HandshakeMessage) GetType() NetworkMessageType {
	return NetworkMessageType_HANDSHAKE
}

func BuildHandshakeMessage(nodeName string) *HandshakeMessage {
	return &HandshakeMessage{
		Body: HandshakeBody{
			NodeName: nodeName,
		},
	}
}

// Implementation of NetworkMessage for HandshakeMessage message.
type StringDataBody string
type StringDataMessage struct {
	Body StringDataBody
}

var _ NetworkMessage = (*StringDataMessage)(nil)

// GetBody implements NetworkMessage.
func (m StringDataMessage) GetBody() any {
	return m.Body
}

// GetType implements NetworkMessage.
func (m StringDataMessage) GetType() NetworkMessageType {
	return NetworkMessageType_STRINGDATA
}

func BuildStringDataMessage(body string) *StringDataMessage {
	return &StringDataMessage{
		Body: StringDataBody(body),
	}
}
