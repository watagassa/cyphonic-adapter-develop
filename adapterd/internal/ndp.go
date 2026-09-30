package internal

import (
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ndp"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
)

// NDPElement
type QueueNDPElement struct {
	sync.Mutex
	buffer *[MaxMessageSize]byte
	packet []byte
	child  *ChildDevice
}

func (adapter *AdapterDevice) NewNDPElement() *QueueNDPElement {
	elem := adapter.GetNDPElement()
	elem.buffer = adapter.GetMessageBuffer()
	elem.Mutex = sync.Mutex{}

	return elem
}

func (elem *QueueNDPElement) clearPointers() {
	elem.buffer = nil
	elem.packet = nil
}

func (adapter *AdapterDevice) NDPStagePacket(elem *QueueNDPElement) {
	for {
		select {
		case adapter.adapterQueue.ndpStaged <- elem:
			return
		default:
		}
		select {
		case tooOld := <-adapter.adapterQueue.ndpStaged:
			// 全て消去
			adapter.PutMessageBuffer(tooOld.buffer)
			adapter.PutNDPElement(tooOld)
		default:
		}
	}
}

func (adapter *AdapterDevice) NDPSendStagePackets() {
	for {
		select {
		case elem := <-adapter.adapterQueue.ndpStaged:
			elem.Lock()
			adapter.adapterQueue.ndp.c <- elem
		default:
			return
		}
	}
}

func (adapter *AdapterDevice) RoutineNDP() {
	defer func() {
		adapter.state.stopping.Done()
		logger.Debug("Routine: NDP - stopped")
	}()
	logger.Debug("Routine: NDP - started")

	for elem := range adapter.adapterQueue.ndp.c {
		packet := gopacket.NewPacket(elem.packet, golayers.LayerTypeEthernet, gopacket.Default)
		etherLayer := packet.Layer(golayers.LayerTypeEthernet)
		srcHWAddr := etherLayer.(*golayers.Ethernet).SrcMAC

		ipv6Layer := packet.Layer(golayers.LayerTypeIPv6)
		srcAddr := netip.MustParseAddr(ipv6Layer.(*golayers.IPv6).SrcIP.String())

		icmpv6Layer := packet.Layer(golayers.LayerTypeICMPv6)
		icmpv6 := icmpv6Layer.(*golayers.ICMPv6)

		switch icmpv6.TypeCode >> 8 {
		case golayers.ICMPv6TypeRouterSolicitation:
			logger.Debug(fmt.Sprintf("Receive Router Solicitation from %v", srcAddr.String()))
			if srcAddr == adapter.InternalInterface.LinkLocalAddr {
				break
			}

			adapter.children.RLock()
			child, ok := adapter.children.keyMap[srcHWAddr.String()]
			adapter.children.RUnlock()
			if !ok {
				break
			}

			child.addrMu.Lock()
			child.LinkLocalAddr = srcAddr
			child.addrMu.Unlock()

			if ns, err := ndp.GenerateNeighborSolicitation(
				adapter.InternalInterface.LinkLocalAddr,
				adapter.InternalInterface.Card.HardwareAddr,
				srcAddr,
				srcHWAddr,
			); err != nil {
				logger.Error(fmt.Errorf("failed to generate Neighbor Solicitation: %w", err))
			} else {
				if err := SendEtherPacket(adapter.socket.receiveIPv6, ns); err != nil {
					logger.Error(fmt.Errorf("failed to send Neighbor Solicitation: %w", err))
				} else {
					logger.Info(fmt.Sprintf("Send Neighbor Solicitation to %v", srcAddr))
				}
			}

			if ra, err := ndp.GenerateRouterAdvertise(
				adapter.InternalInterface.LinkLocalAddr,
				adapter.InternalInterface.Card.HardwareAddr,
				srcAddr,
			); err != nil {
				logger.Error(fmt.Errorf("failed to generate Router Advertisement: %w", err))
			} else {
				if err := SendEtherPacket(adapter.socket.receiveIPv6, ra); err != nil {
					logger.Error(fmt.Errorf("failed to send Router Advertisement: %w", err))
				} else {
					logger.Info(fmt.Sprintf("Send Router Advertisement to %v", srcAddr))
				}
			}
		case golayers.ICMPv6TypeRouterAdvertisement:
			logger.Debug(fmt.Sprintf("Receive Router Advertisement from %v", srcAddr.String()))
		case golayers.ICMPv6TypeNeighborSolicitation:
			logger.Info(fmt.Sprintf("Receive Neighbor Solicitation from %v", srcAddr.String()))
			target, ok := netip.AddrFromSlice(icmpv6.LayerContents()[8:24])
			if !ok {
				logger.Error(fmt.Errorf("parse error: %v", icmpv6.LayerContents()[8:24]))
			}
			if err := adapter.DoNDProxy(target, srcAddr, srcHWAddr); err != nil {
				logger.Error(fmt.Errorf("failed to execution NDProxy: %w", err))
			}
		case golayers.ICMPv6TypeNeighborAdvertisement:
			logger.Debug(fmt.Sprintf("Receive Neighbor Advertisement from %v", srcAddr))
		}
		adapter.PutMessageBuffer(elem.buffer)
		adapter.PutNDPElement(elem)
	}
}

// DoNDProxy executes ND Proxy.
// RFC4389
// *** Regulation ***
// - TargetAddress is not all LLA.
// - SourceAddress is not adapter's LLA.
// - SourceAddress is not ::.
func (adapter *AdapterDevice) DoNDProxy(targetAddr, srcAddr netip.Addr, srcHWAddr net.HardwareAddr) error {
	if !targetAddr.IsLinkLocalUnicast() && srcAddr != adapter.InternalInterface.LinkLocalAddr && srcAddr != netip.MustParseAddr(ip.IPv6Unspecified) {
		if na, err := ndp.GenerateNeighborAdvertisement(
			adapter.InternalInterface.LinkLocalAddr,
			adapter.InternalInterface.Card.HardwareAddr,
			targetAddr,
			srcAddr,
			srcHWAddr,
		); err != nil {
			return fmt.Errorf("failed to generate Neighbor Advertisement: %w", err)
		} else {
			if err := SendEtherPacket(adapter.socket.receiveIPv6, na); err != nil {
				return fmt.Errorf("failed to send Neighbor Advertisement: %w", err)
			}
			logger.Info(fmt.Sprintf("Send Neighbor Advertisement to %v", srcAddr))
		}
	}

	return nil
}
