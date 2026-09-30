// Package layers contains the layer structure of CYPHONIC Packet.
package layers

// BasePacket is a convenience struct which implements the PacketData and
// PacketPayload and PacketHMAC functions of the Packet interface.
type BasePacket struct {
	// BaseHeader is the set of bytes that make up this packet.
	BaseHeader []byte
	// Payload is the set of bytes contained by (but not part of) this
	// Layer.  Again, to take Ethernet as an example, this would be the
	// set of bytes encapsulated by the Ethernet protocol.
	Payload []byte
	// HMAC is Hash-based Message Authentication Code of a CYPHONIC packet.
	HMAC []byte
}

// PacketBaseHeader returns the bytes of the packet.
func (b *BasePacket) PacketBaseHeader() []byte { return b.BaseHeader }

// PacketPayload returns the payload contained within the packet.
func (b *BasePacket) PacketPayload() []byte { return b.Payload }

// PacketHMAC returns the HMAC contained within the packet.
func (b *BasePacket) PacketHMAC() []byte { return b.HMAC }

// NodeID offset length.
const (
	MessageOffsetNodeID = 16
	NodeIDlen           = 16
)

// PathID offset length.
const (
	MessageOffsetPathID = 16
	PathIDlen           = 16
)

// PacketType offset length.
const MessageOffsetPacketType = 6
