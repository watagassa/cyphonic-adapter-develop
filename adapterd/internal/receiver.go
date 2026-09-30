package internal

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// QueueInboundElement is buffer.
type QueueInboundElement struct {
	sync.Mutex
	buffer *[MaxMessageSize]byte
	packet []byte
	key    []byte
}

// clearPointers clears elem fields that contain pointers.
// This makes the garbage collector's life easier and
// avoids accidentally keeping other objects around unnecessarily.
// It also reduces the possible collateral damage from use-after-free bugs.
func (elem *QueueInboundElement) clearPointers() {
	elem.buffer = nil
	elem.packet = nil
	elem.key = nil
}

// ChildRoutineSequentialReceiver writes adapter internal network interface.
// Write packet using raw sockets.
func (peer *Peer) ChildRoutineSequentialReceiver() {
	child := peer.child
	defer func() {
		peer.stopping.Done()
		logger.Debug(fmt.Sprintf("Routine: Sequential Receiver - stopped: [child name=%s] [peer vip4=%v] [peer vip6=%v]", child.deviceName, peer.dstVirtualIPv4, peer.dstVirtualIPv6))
	}()
	logger.Debug(fmt.Sprintf("Routine: Sequential Receiver - started: [child name=%s]", child.deviceName))

	for elem := range peer.queue.inbound.c {
		if elem == nil {
			return
		}

		elem.Lock()
		if elem.packet == nil {
			// decryption failed
			goto skip
		}

		if len(elem.packet) == 0 {
			goto skip
		}

		switch elem.packet[0] >> 4 {
		case ipv4.Version:
			if err := SendPacket4(peer.child.Adapter.socket.sendIPv4, elem.packet, elem.packet[ip.IPv4offsetDst:ip.IPv4offsetDst+net.IPv4len]); err != nil {
				logger.Error(fmt.Errorf("failed to send capsule message to %v: %w", elem.packet[ip.IPv4offsetDst:ip.IPv4offsetDst+net.IPv4len], err))
			}
		case ipv6.Version:
			if err := SendPacket6(peer.child.Adapter.socket.sendIPv6, elem.packet, elem.packet[ip.IPv6offsetDst:ip.IPv6offsetDst+net.IPv6len]); err != nil {
				logger.Error(fmt.Errorf("failed to send capsule message to %v: %w", elem.packet[ip.IPv6offsetDst:ip.IPv6offsetDst+net.IPv6len], err))
			}
		default:
			logger.Error("ip version error")
		}

	skip:
		child.Adapter.PutMessageBuffer(elem.buffer)
		child.Adapter.PutInboundElement(elem)
	}
}

// RoutineDecryption dencrypts packets to use go routine.
func (adapter *AdapterDevice) RoutineDecryption() {
	defer func() {
		adapter.queue.encryption.wg.Done()
		logger.Debug("Routine: Decryption worker - stopped")
	}()
	logger.Debug("Routine: Decryption worker - start")

	for elem := range adapter.queue.decryption.c {
		encryptedLen := binary.BigEndian.Uint16(elem.packet[12:14]) - 16
		pb, err := layers.DecryptPacket(elem.packet[32:encryptedLen], elem.key)
		if err != nil {
			logger.Error(fmt.Errorf("failed to decrypt message: %w", err))
		}

		salt := layers.InitSalt(elem.packet)
		payloadLen := 4 + binary.BigEndian.Uint16(pb[0:2])

		if ok := layers.DecodeHMAC(elem.packet[encryptedLen:encryptedLen+16], elem.packet[0:encryptedLen], salt); !ok {
			logger.Error(fmt.Errorf("failed to decode capsule message's HMAC: %w", err))
			elem.packet = nil
		} else {
			elem.packet = pb[4:payloadLen]
		}

		elem.Unlock()
	}
}
