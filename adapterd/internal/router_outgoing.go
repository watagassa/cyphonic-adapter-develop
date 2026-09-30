package internal

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"syscall"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ether"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// RoutineSendOutgoing reads buffers from adapter's internal network interface to use go routine.
// The internal interface connects with multiple general nodes.
// Received packets are stored in respective staging queues.
// - Adapter's packet flow: General node -> Internal interface -> External interface -> Peer node
func (adapter *AdapterDevice) RoutineSendOutgoing() {
	defer func() {
		adapter.state.stopping.Done()
		adapter.queue.encryption.wg.Done()
		logger.Debug("Routine: Send Outgoing - stopped")
	}()
	logger.Debug("Routine: Send Outgoing - started")

	var elem *QueueOutboundElement
	var arpelem *QueueARPElement
	var dnselem *QueueDNSElement
	var ndpelem *QueueNDPElement

	for {
		if elem != nil {
			// return buffer to pool
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutOutboundElement(elem)
		}
		elem = adapter.NewOutboundElement()

		if arpelem != nil {
			// return buffer to pool
			adapter.PutMessageBuffer(arpelem.buffer)
			adapter.PutARPElement(arpelem)
		}
		arpelem = adapter.NewARPElement()

		if dnselem != nil {
			// return buffer to pool
			adapter.PutMessageBuffer(dnselem.buffer)
			adapter.PutDNSElement(dnselem)
		}
		dnselem = adapter.NewDNSElement()

		if ndpelem != nil {
			// return buffer to pool
			adapter.PutMessageBuffer(ndpelem.buffer)
			adapter.PutNDPElement(ndpelem)
		}
		ndpelem = adapter.NewNDPElement()

		var size int
		var err error

		// read packet
		if adapter.InternalIPVersion == layers.TypeLocalIPVersion4 {
			size, err = syscall.Read(adapter.socket.receiveIPv4, elem.buffer[:])
		} else if adapter.InternalIPVersion == layers.TypeLocalIPVersion6 {
			size, err = syscall.Read(adapter.socket.receiveIPv6, elem.buffer[:])
		} else {
			logger.Error(fmt.Errorf("unsupported IP version: %d", adapter.InternalIPVersion))
			return
		}

		if err != nil {
			// return buffer to pool
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutOutboundElement(elem)
			adapter.PutMessageBuffer(arpelem.buffer)
			adapter.PutARPElement(arpelem)
			adapter.PutMessageBuffer(dnselem.buffer)
			adapter.PutDNSElement(dnselem)
			adapter.PutMessageBuffer(ndpelem.buffer)
			adapter.PutNDPElement(ndpelem)
			return
		}

		if size == 0 {
			continue
		}

		/* Handle with Ether layer */
		eh := &ether.EtherHeader{
			DstMacAddr: elem.buffer[ether.EtheroffsetDst : ether.EtheroffsetDst+ether.DstMacLength],
			SrcMacAddr: elem.buffer[ether.EtheroffsetSrc : ether.EtheroffsetSrc+ether.SrcMacLength],
			ProtoType:  binary.BigEndian.Uint16(elem.buffer[ether.EtheroffsetProtocolTypeLength : ether.EtheroffsetProtocolTypeLength+ether.ProtocolTypeLength]),
		}
		packet := gopacket.NewPacket(elem.buffer[ether.Etheroffset:size], golayers.LayerTypeEthernet, gopacket.Default)

		// Find general node cache information.
		child, ok := adapter.children.keyMap[eh.SrcMacAddr.String()]
		if !ok {
			continue
		}

		/* Handle with IP layer */
		elem.packet = elem.buffer[ether.IPoffset:size]

		var peer *Peer

		switch eh.ProtoType {
		case ether.EthTypeIPv4:
			if len(elem.packet) < ipv4.HeaderLen {
				logger.Warn(fmt.Sprintf("received packet too small=%d", len(elem.packet)))
				continue
			}
			dst := elem.packet[ip.IPv4offsetDst : ip.IPv4offsetDst+net.IPv4len]
			peer, err = child.allowedips.Lookup(dst)
			if err != nil {
				logger.Warn("cannot find child device")
			}

			udpLayer := packet.Layer(golayers.LayerTypeUDP)
			if udpLayer != nil {
				udp := udpLayer.(*golayers.UDP)
				switch udp.DstPort.LayerType() {
				case golayers.LayerTypeDNS:
					dnselem.packet = elem.packet // IP layer -
					dnselem.udp = udp
					dnselem.child = child
					elem = nil // Warning: If not removed here, the elem.buffer passed to the DNSProcessStage may be overwritten.

					// Discards DNS requests when authentication status is incomplete or child device is not running.
					if dnselem.child.isLogin.Get() && dnselem.child.isUp() {
						adapter.DNSProcessStage(dnselem)
					} else {
						logger.Warn(fmt.Sprintf("%s is unauthenticated or has a closed state", child.deviceName))
					}
					dnselem = nil
					continue
				}
			}

			if peer != nil && peer.isRunning.Get() {
				peer.StagePacket(elem)
				elem = nil
				peer.SendStagedPackets()
			}
		case ether.EthTypeIPv6:
			if len(elem.packet) < ipv6.HeaderLen {
				logger.Warn(fmt.Sprintf("received packet too small=%d", len(elem.packet)))
				continue
			}

			// srcAddr, ok := netip.AddrFromSlice(elem.packet[ip.IPv6offsetSrc : ip.IPv6offsetSrc+net.IPv6len])
			// if !ok {
			// 	logger.Error(fmt.Errorf("parse error: %v", elem.packet[ip.IPv6offsetSrc:ip.IPv6offsetSrc+net.IPv6len]))
			// }

			dstAddr, ok := netip.AddrFromSlice(elem.packet[ip.IPv6offsetDst : ip.IPv6offsetDst+net.IPv6len])
			if !ok {
				logger.Error(fmt.Errorf("parse error: %v", elem.packet[ip.IPv6offsetDst:ip.IPv6offsetDst+net.IPv6len]))
			}

			peer, err = child.allowedips.Lookup(net.IP(dstAddr.AsSlice()).To16())
			if err != nil {
				logger.Warn("cannot find child device")
			}

			icmpv6Layer := packet.Layer(golayers.LayerTypeICMPv6)
			if icmpv6Layer != nil {
				icmpv6 := icmpv6Layer.(*golayers.ICMPv6)
				switch icmpv6.TypeCode.Type() {
				case golayers.ICMPv6TypeDestinationUnreachable:
				case golayers.ICMPv6TypeEchoRequest:
				case golayers.ICMPv6TypeEchoReply:
				case golayers.ICMPv6TypeRouterSolicitation, golayers.ICMPv6TypeRouterAdvertisement, golayers.ICMPv6TypeNeighborSolicitation, golayers.ICMPv6TypeNeighborAdvertisement:
					ndpelem.buffer = elem.buffer // Ether layer~
					ndpelem.packet = elem.buffer[ether.Etheroffset:size]
					ndpelem.child = child
					elem = nil
					adapter.NDPStagePacket(ndpelem)
					ndpelem = nil
					adapter.NDPSendStagePackets()
				case golayers.ICMPv6TypeMLDv2MulticastListenerReportMessageV2:
				default:
					logger.Error(fmt.Errorf("Unknown ICMPv6 type: %v", icmpv6.TypeCode.Type()))
				}
			}

			udpLayer := packet.Layer(golayers.LayerTypeUDP)
			if udpLayer != nil {
				udp := udpLayer.(*golayers.UDP)
				switch udp.DstPort.LayerType() {
				case golayers.LayerTypeDNS:
					dnselem.packet = elem.packet // IP layer~
					dnselem.udp = udp
					dnselem.child = child
					elem = nil // Warning: If not removed here, the elem.buffer passed to the DNSProcessStage may be overwritten.
					adapter.DNSProcessStage(dnselem)
					dnselem = nil
				default:
				}
			}

			if peer != nil && peer.isRunning.Get() && dstAddr != adapter.InternalInterface.Addr6 && dstAddr != adapter.child.VirtualIPv6 {
				// Do not delete.
				// : if peer != nil && peer.isRunning.Get() && !bytes.Equal(dst, ADAPTER_IPV6_ADDRESS) && !bytes.Equal(dst, device.gnDevice.AssignVirtualIPv6) {
				peer.StagePacket(elem)
				elem = nil
				peer.SendStagedPackets()
			}
		case ether.EthTypeARP:
			arpelem.buffer = elem.buffer // Ether layer~
			arpelem.packet = elem.buffer[ether.Etheroffset:size]
			arpelem.child = child
			elem = nil
			adapter.ARPStagePacket(arpelem)
			arpelem = nil
			adapter.ARPSendStagePackets()
		case ether.EthTypeRARP:
			logger.Warn("non-supported message: Reverse Address Resolution Protocol (RARP)")
		case ether.EthTypeVMTP:
			logger.Warn("non-supported message: Versatile Message Transaction Protocol (VMTP)")
		case ether.EthTypeAppleTalk:
			logger.Warn("non-supported message: AppleTalk (EtherTalk)")
		case ether.EthTypeAARP:
			logger.Warn("non-supported message: AppleTalk Address Resolution Porotocol (AARP)")
		case ether.EthTypeIPX:
			logger.Warn("non-supported message: Novell Netware")
		case ether.EthTypeSNMP:
			logger.Warn("non-supported message: Simple Network Management Protocol (SNMP) over Ethernet")
		case ether.EthTypeNetBIOS:
			logger.Warn("non-supported message: NetBIOS/NetBEUI")
		case ether.EthTypeXTP:
			logger.Warn("non-supported message: Xpress Transport Protocol (XTP)")
		case ether.EthTypePPPoEDiscoveryStage:
			logger.Warn("non-supported message: PPPoE Discovery Stage")
		case ether.EthTypePPPoESessionStage:
			logger.Warn("non-supported message: PPPoE Session Stage")
		case ether.EthTypeRRCP:
			logger.Warn("non-supported message: Realtek Remote Control Protocol")
		case ether.EthTypeLoopDetection:
			logger.Warn("loop detection")
		default:
			logger.Error(fmt.Errorf("Detect unknown protocol version: %v", eh.ProtoType))
		}
	}
}
