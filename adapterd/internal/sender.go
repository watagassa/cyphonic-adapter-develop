package internal

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// QueueOutboundElement is buffer.
type QueueOutboundElement struct {
	sync.Mutex
	buffer *[MaxMessageSize]byte // slice holding the packet data
	packet []byte                // slice of "buffer" (always!)
	peer   *Peer                 // related peer
}

// NewOutboundElement initializes Outbound element.
func (adapter *AdapterDevice) NewOutboundElement() *QueueOutboundElement {
	elem := adapter.GetOutboundElement()
	elem.buffer = adapter.GetMessageBuffer()
	elem.Mutex = sync.Mutex{}

	return elem
}

// clearPointers clears elem fields that contain pointers.
// This makes the garbage collector's life easier and
// avoids accidentally keeping other objects around unnecessarily.
// It also reduces the possible collateral damage from use-after-free bugs.
func (elem *QueueOutboundElement) clearPointers() {
	elem.buffer = nil
	elem.packet = nil
}

// StagePacket stating peer queue from outboundElement.
func (peer *Peer) StagePacket(elem *QueueOutboundElement) {
	for {
		select {
		case peer.queue.staged <- elem:
			return
		default:
		}
		select {
		case tooOld := <-peer.queue.staged:
			peer.child.Adapter.PutMessageBuffer(tooOld.buffer)
			peer.child.Adapter.PutOutboundElement(tooOld)
		default:
		}
	}
}

// SendStagedPackets staging outbound queue and encryption queue.
// CapsuleMessage（ICMP Request）の生成 + 暗号化処理
func (peer *Peer) SendStagedPackets() {
	for {
		select {
		case elem := <-peer.queue.staged:
			elem.peer = peer

			elem.Lock()

			if peer.isRunning.Get() {
				peer.queue.outbound.c <- elem                 // Sender routine
				peer.child.Adapter.queue.encryption.c <- elem // Adapter's processing routine
			} else {
				peer.child.Adapter.PutMessageBuffer(elem.buffer)
				peer.child.Adapter.PutOutboundElement(elem)
			}
		default:
			return
		}
	}
}

// ChildRoutineSequentialSender reads packets from sending queue and sends to endpoint.
// Write packet using standard sockets.
func (peer *Peer) ChildRoutineSequentialSender() {
	child := peer.child
	defer func() {
		peer.stopping.Done()
		logger.Debug(fmt.Sprintf("Routine: Sequential Sender - stopped: [child name=%s] [peer vip4=%v] [peer vip6=%v]", child.deviceName, peer.dstVirtualIPv4, peer.dstVirtualIPv6))
	}()
	logger.Debug(fmt.Sprintf("Routine: Sequential Sender - started: [child name=%s]", child.deviceName))

	for elem := range peer.queue.outbound.c {
		if elem == nil {
			return
		}
		elem.Lock()
		if !peer.isRunning.Get() {
			child.Adapter.PutMessageBuffer(elem.buffer)
			child.Adapter.PutOutboundElement(elem)
			continue
		}

		// send message and return buffer to pool
		if peer.isTRS.Get() {
			switch child.Adapter.ExternalIPVersion {
			case layers.TypeLocalIPVersion4:
				_, err := peer.connUDP.WriteToUDPAddrPort(elem.packet, peer.endpointTRSv4.AddrPort())
				child.Adapter.PutMessageBuffer(elem.buffer)
				child.Adapter.PutOutboundElement(elem)
				if err != nil {
					continue
				}
			case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
				_, err := peer.connUDP.WriteToUDPAddrPort(elem.packet, peer.endpointTRSv6.AddrPort())
				child.Adapter.PutMessageBuffer(elem.buffer)
				child.Adapter.PutOutboundElement(elem)
				if err != nil {
					continue
				}
			}
		} else {
			switch child.Adapter.ExternalIPVersion {
			case layers.TypeLocalIPVersion4:
				_, err := peer.connUDP.WriteToUDPAddrPort(elem.packet, peer.endpointv4.AddrPort())

				child.Adapter.PutMessageBuffer(elem.buffer)
				child.Adapter.PutOutboundElement(elem)
				if err != nil {
					continue
				}
			case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
				_, err := peer.connUDP.WriteToUDPAddrPort(elem.packet, peer.endpointv6.AddrPort())
				if err != nil {
					continue
				}
			}
		}
	}
}

// RoutineEncryption encrypts packets to use go routine.
func (adapter *AdapterDevice) RoutineEncryption() {
	defer func() {
		adapter.queue.encryption.wg.Done()
		logger.Debug("Routine: Encryption worker - stopped")
	}()
	logger.Debug("Routine: Encryption worker - start")

	for elem := range adapter.queue.encryption.c {
		var pathID string

		cm := layers.CapsuleMessage{
			BaseHeader:    layers.BaseHeader{},
			PayloadLength: 0,
			Padding:       0,
			Payload:       []byte{},
			HMAC:          []byte{},
		}

		err := layers.SerializeBaseHeader(&cm.BaseHeader)
		if err != nil {
			logger.Error(fmt.Errorf("failed to serialize BaseHeader: %w", err))
		}

		layers.SerializeType(&cm.BaseHeader, layers.TypeClassCapsuleMessage)
		cm.PayloadLength = uint16(len(elem.packet))
		cm.Payload = elem.packet

		pathID = hex.EncodeToString(elem.peer.pathID)
		hexID, err := hex.DecodeString(pathID)
		if err != nil {
			logger.Error(fmt.Errorf("failed to convert strig to byte in PathID: %w", err))
		}
		cm.BaseHeader.ID = hexID

		adapter.child.peers.Lock()

		if peer, ok := elem.peer.child.peers.keyMap[pathID]; ok {
			if peer.nodeType.Get() {
				layers.SerializeFlag(&cm.BaseHeader, layers.FlagClassNodeTypeMN)
			} else {
				layers.SerializeFlag(&cm.BaseHeader, layers.FlagClassNodeTypeCN)
			}
			endKey := peer.endKey
			packet, err := cm.Marshal(endKey)
			if err != nil {
				logger.Error(fmt.Errorf("failed to marshal Capsule Message: %w", err))
			}

			elem.packet = packet
		} else {
			elem.packet = nil
		}
		adapter.child.peers.Unlock()

		elem.Unlock()
	}
}
