package internal

import (
	"errors"
	"fmt"
	"net/netip"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/arp"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
)

// ARPElement
type QueueARPElement struct {
	sync.Mutex
	buffer *[MaxMessageSize]byte
	packet []byte
	child  *ChildDevice
}

func (adapter *AdapterDevice) NewARPElement() *QueueARPElement {
	elem := adapter.GetARPElement()
	elem.buffer = adapter.GetMessageBuffer()
	elem.Mutex = sync.Mutex{}

	return elem
}

func (elem *QueueARPElement) clearPointers() {
	elem.buffer = nil
	elem.packet = nil
}

func (adapter *AdapterDevice) ARPStagePacket(elem *QueueARPElement) {
	for {
		select {
		case adapter.adapterQueue.arpStaged <- elem:
			return
		default:
		}
		select {
		case tooOld := <-adapter.adapterQueue.arpStaged:
			// 全て消去
			adapter.PutMessageBuffer(tooOld.buffer)
			adapter.PutARPElement(tooOld)
		default:
		}
	}
}

func (adapter *AdapterDevice) ARPSendStagePackets() {
	for {
		select {
		case elem := <-adapter.adapterQueue.arpStaged:
			elem.Lock()
			adapter.adapterQueue.arp.c <- elem
		default:
			return
		}
	}
}

func (adapter *AdapterDevice) RoutineARP() {
	defer func() {
		adapter.state.stopping.Done()
		logger.Debug("Routine: ARP - stopped")
	}()
	logger.Debug("Routine: ARP - started")

	for elem := range adapter.adapterQueue.arp.c {
		packet := gopacket.NewPacket(elem.packet, golayers.LayerTypeEthernet, gopacket.Default)
		etherLayer := packet.Layer(golayers.LayerTypeARP)
		if etherLayer != nil {
			areq, ok := etherLayer.(*golayers.ARP)
			if !ok {
				logger.Error("parse error")
			}

			shouldProxyArp, targetAddr, err := shouldProxyARP(areq, elem.child)
			if err != nil {
				logger.Error(fmt.Errorf("failed to get ARP Target Address: %w", err))
			}

			if shouldProxyArp {
				ares := arp.GenerateARPReply(areq, adapter.InternalInterface.Card.HardwareAddr)
				if err := SendEtherPacket(adapter.socket.receiveIPv4, ares); err != nil {
					logger.Error(fmt.Errorf("failed to send ARP Response to %v: %w", adapter.socket.receiveIPv4, err))
				} else {
					// return buffer to pool
					adapter.PutMessageBuffer(elem.buffer)
					adapter.PutARPElement(elem)

					logger.Debug(fmt.Sprintf("Executed ProxyARP: Target Address=%v MAC Address=%v", targetAddr, adapter.InternalInterface.Card.HardwareAddr))
				}
			} else {
				// return buffer to pool
				adapter.PutMessageBuffer(elem.buffer)
				adapter.PutARPElement(elem)
			}
		}
	}
}

// shouldProxyARP determines whether to run ProxyARP.
// Excludes ARP for server (adapter) IP and ARP for general node own IP.
/*
  ProxyARP execution condition
  - conditions 1: ARP target address is not own (child) virtual IP address.
  - conditions 2: ARP target address is not adapter's virtual IP address.
  - conditions 3: ARP target address is not a broadcast address.
  - conditions 4: ARP target address is not a link-local address.
  - conditions 5: child device is aliving.
*/
func shouldProxyARP(areq *golayers.ARP, child *ChildDevice) (bool, netip.Addr, error) {
	target, ok := netip.AddrFromSlice(areq.DstProtAddress)
	if !ok {
		err := errors.New("parse error")
		return false, netip.Addr{}, err
	}

	return target != child.VirtualIPv4 && target != child.Adapter.InternalInterface.Addr4 && !ip.IsBroadcastAddress(target) && !target.IsLinkLocalUnicast() && child.isUp(), target, nil
}
