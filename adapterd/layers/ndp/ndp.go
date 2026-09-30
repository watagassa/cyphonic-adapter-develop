package ndp

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ether"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
)

// GenerateRouterSolicitation generates Router Solicitation packet.
// RS: 133 => src: LLA / dst: ff02::2
// rfc4861
// Hosts send Router Solicitations in order to prompt routers to generate
// Router Advertisements quickly.
// Source Address An IP address assigned to the sending interface, or the
// unspecified address if no address is assigned to the sending interface.
// Destination Address Typically the all-routers multicast address.
func GenerateRouterSolicitation(localAddr netip.Addr, localHWAddr net.HardwareAddr) ([]byte, error) {
	dstMacAddr, err := net.ParseMAC(ether.AllRouterMacAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MAC address: %w", err)
	}

	eth := golayers.Ethernet{
		SrcMAC:       localHWAddr,
		DstMAC:       dstMacAddr,
		EthernetType: golayers.EthernetTypeIPv6,
	}

	ipv6 := golayers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       0,
		NextHeader:   golayers.IPProtocolICMPv6,
		HopLimit:     255,
		SrcIP:        net.IP(localAddr.AsSlice()).To16(),
		DstIP:        net.ParseIP(ip.AllRouterMulticastAddress).To16(),
	}

	icmpv6 := golayers.ICMPv6{
		TypeCode: golayers.ICMPv6TypeRouterSolicitation << 8,
	}

	rs := golayers.ICMPv6RouterSolicitation{
		Options: []golayers.ICMPv6Option{
			{
				Type: golayers.ICMPv6OptSourceAddress,
				Data: localHWAddr,
			},
		},
	}

	if err := icmpv6.SetNetworkLayerForChecksum(&ipv6); err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	if err := gopacket.SerializeLayers(buffer, options,
		&eth,
		&ipv6,
		&icmpv6,
		&rs,
	); err != nil {
		return nil, fmt.Errorf("failed to serialize Router Solicitation packet: %w", err)
	}

	outgoingPacket := buffer.Bytes()

	return outgoingPacket, nil
}

// GenerateRouterAdvertise generates Router Advertise packet.
// RA: 134 => src: LLA / dst: LLA or ULA or ff02::1
// rfc4861
// Routers send out Router Advertisement messages periodically, or in
// response to Router Solicitations.
// Source Address MUST be the link-local address assigned to the interface
// from which this message is sent.
// Destination Address Typically the Source Address of an invoking Router
// Solicitation or the all-nodes multicast address.
func GenerateRouterAdvertise(localAddr netip.Addr, localHWAddr net.HardwareAddr, dstAddr netip.Addr) ([]byte, error) {
	dstMacAddr, err := net.ParseMAC(ether.AllNodeMacAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MAC address: %w", err)
	}

	eth := golayers.Ethernet{
		SrcMAC:       localHWAddr,
		DstMAC:       dstMacAddr,
		EthernetType: golayers.EthernetTypeIPv6,
	}

	ipv6 := golayers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       0,
		NextHeader:   golayers.IPProtocolICMPv6,
		HopLimit:     255,
		SrcIP:        net.IP(localAddr.AsSlice()).To16(),
		DstIP:        net.IP(dstAddr.AsSlice()).To16(),
	}

	icmpv6 := golayers.ICMPv6{
		TypeCode: golayers.ICMPv6TypeRouterAdvertisement << 8,
	}

	ra := golayers.ICMPv6RouterAdvertisement{
		HopLimit:       64,
		Flags:          0xc0, // M: 1 / O: 1
		RouterLifetime: 30,
		ReachableTime:  0,
		RetransTimer:   0,
		Options: []golayers.ICMPv6Option{
			{
				Type: golayers.ICMPv6OptSourceAddress,
				Data: localHWAddr,
			},
		},
	}

	if err := icmpv6.SetNetworkLayerForChecksum(&ipv6); err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	if err := gopacket.SerializeLayers(buffer, options,
		&eth,
		&ipv6,
		&icmpv6,
		&ra,
	); err != nil {
		return nil, fmt.Errorf("failed to serialize Router Advertise packet: %w", err)
	}

	outgoingPacket := buffer.Bytes()

	return outgoingPacket, nil
}

// GenerateNeighborSolicitation generates Neighbor Solicitation packet.
// NA: 135 => src: LLA / dst: LLA or ULA
// https://www.n-study.com/ipv6-detail/ipv6-address-resolution/
// rfc4861
// Source Address Either an address assigned to the interface from which
// this message is sent or (if Duplicate Address Detection is in progress
// [ADDRCONF]) the unspecified address. Destination Address Either the
// solicited-node multicast address corresponding to the target address, or
// the target address. Hop Limit 255
func GenerateNeighborSolicitation(localAddr netip.Addr, localHWAddr net.HardwareAddr, dstAddr netip.Addr, dstHWAddr net.HardwareAddr) ([]byte, error) {
	eth := golayers.Ethernet{
		SrcMAC:       localHWAddr,
		DstMAC:       dstHWAddr,
		EthernetType: golayers.EthernetTypeIPv6,
	}

	ipv6 := golayers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       0,
		NextHeader:   golayers.IPProtocolICMPv6,
		HopLimit:     255,
		SrcIP:        net.IP(localAddr.AsSlice()).To16(),
		DstIP:        net.IP(dstAddr.AsSlice()).To16(),
	}

	icmpv6 := golayers.ICMPv6{
		TypeCode: golayers.ICMPv6TypeNeighborSolicitation << 8,
	}

	ns := golayers.ICMPv6NeighborSolicitation{
		TargetAddress: net.IP(dstAddr.AsSlice()).To16(),
		Options: []golayers.ICMPv6Option{
			{
				Type: golayers.ICMPv6OptSourceAddress,
				Data: localHWAddr,
			},
		},
	}

	if err := icmpv6.SetNetworkLayerForChecksum(&ipv6); err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	if err := gopacket.SerializeLayers(buffer, options,
		&eth,
		&ipv6,
		&icmpv6,
		&ns,
	); err != nil {
		return nil, fmt.Errorf("failed to selialize Neighbor Solicitation: %w", err)
	}

	outgoingPacket := buffer.Bytes()

	return outgoingPacket, nil
}

// GenerateNeighborAdvertisement generates Neighbor Advertisement packet.
// NA: 136 => src: LLA / dst: LLA or ULA
// https://www.n-study.com/ipv6-detail/ipv6-address-resolution/
// rfc4861
// Source Address An address assigned to the interface from which the
// advertisement is sent. Destination Address For solicited advertisements,
// the Source Address of an invoking Neighbor Solicitation or, if the
// solicitation's Source Address is the unspecified address, the all-nodes
// multicast address.
func GenerateNeighborAdvertisement(localAddr netip.Addr, localHWAddr net.HardwareAddr, targetAddr netip.Addr, dstAddr netip.Addr, dstHWAddr net.HardwareAddr) ([]byte, error) {
	eth := golayers.Ethernet{
		SrcMAC:       localHWAddr,
		DstMAC:       dstHWAddr,
		EthernetType: golayers.EthernetTypeIPv6,
	}

	ipv6 := golayers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       0,
		NextHeader:   golayers.IPProtocolICMPv6,
		HopLimit:     255,
		SrcIP:        net.IP(localAddr.AsSlice()).To16(),
		DstIP:        net.IP(dstAddr.AsSlice()).To16(),
	}

	icmpv6 := golayers.ICMPv6{
		TypeCode: golayers.ICMPv6TypeNeighborAdvertisement << 8,
	}

	na := golayers.ICMPv6NeighborAdvertisement{
		Flags:         0x20 | 0x40, // solicited && override
		TargetAddress: net.IP(targetAddr.AsSlice()).To16(),
		Options: []golayers.ICMPv6Option{
			{
				Type: golayers.ICMPv6OptTargetAddress,
				Data: localHWAddr,
			},
		},
	}

	if err := icmpv6.SetNetworkLayerForChecksum(&ipv6); err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	if err := gopacket.SerializeLayers(buffer, options,
		&eth,
		&ipv6,
		&icmpv6,
		&na,
	); err != nil {
		return nil, fmt.Errorf("failed to selialize Neighbor Advertisement: %w", err)
	}

	outgoingPacket := buffer.Bytes()

	return outgoingPacket, nil
}

// GenerateSolicitedNodeMulticastAddress generates Solicited-Node Multicast Address and its corresponding MAC address.
// rfc4291
// The solicited-node multicast address is formed by taking the low-order
// 24 bits of the corresponding unicast or anycast address and appending
// those bits to the prefix FF02:0:0:0:0:1:FF00::/104.
func GenerateSolicitedNodeMulticastAddress(srcAddr netip.Addr, srcHWAddr net.HardwareAddr) (net.IP, net.HardwareAddr, error) {
	if !srcAddr.Is6() {
		return nil, nil, fmt.Errorf("invalid address: %v", srcAddr)
	}

	srcBytes := srcAddr.As16()

	ipPrefix := []byte{0xff, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xff}
	macPrefix := []byte{0x33, 0x33, 0xff}

	solicitedNodeMulticastIPv6Address := append(ipPrefix, srcBytes[13], srcBytes[14], srcBytes[15])

	solicitedNodeMulticastMACAddress := append(macPrefix, srcBytes[13], srcBytes[14], srcBytes[15])

	return net.IP(solicitedNodeMulticastIPv6Address), net.HardwareAddr(solicitedNodeMulticastMACAddress), nil
}
