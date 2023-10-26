package internal

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
)

// Implementation of NetworkMessage for UDPDiscoveryMessage message.
type UDPDiscoveryBody struct {
	Ipv4 string
}
type UDPDiscoveryMessage struct {
	Type NetworkMessageType
	Body UDPDiscoveryBody
}

var _ NetworkMessage = (*UDPDiscoveryMessage)(nil)

func (m UDPDiscoveryMessage) GetType() NetworkMessageType {
	return m.Type
}

func (m UDPDiscoveryMessage) GetBody() any {
	return m.Body
}

func BuildUDPDiscoveryMessage(body UDPDiscoveryBody) *UDPDiscoveryMessage {
	return &UDPDiscoveryMessage{
		Type: NetworkMessageType_UDP_DISCOVERY,
		Body: body,
	}
}
