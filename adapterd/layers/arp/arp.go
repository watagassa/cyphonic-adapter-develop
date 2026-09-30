package arp

import (
	"encoding/binary"
	"net"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ether"
	golayers "github.com/google/gopacket/layers"
)

type LinkType uint8
type EthernetType uint16
type OperationCode uint16

const (
	HardwareSize = 6
	ProtocolSize = 4
)

type ARPHeader struct {
	HardwareType LinkType
	ProtocolType EthernetType
	HardwareSize uint8
	ProtocolSize uint8
	OpCode       OperationCode
}

type ARP struct {
	Header                ARPHeader
	SourceHardwareAddress []byte
	SourceProtocolAddress []byte
	TargetHardwareAddress []byte
	TargetProtocolAddress []byte
}

// Potential values for ARP operation.
const (
	ARPRequest = 1
	ARPReply   = 2
)

// Generate ARP Reply generates an ARP Replay packet.
func GenerateARPReply(req *golayers.ARP, adapterMacAddr net.HardwareAddr) []byte {
	outgoing := make([]byte, 1500)

	copy(outgoing[ether.EtheroffsetDst:ether.EtheroffsetDst+ether.DstMacLength], req.SourceHwAddress)                                                        // Dst mac address
	copy(outgoing[ether.EtheroffsetSrc:ether.EtheroffsetSrc+ether.SrcMacLength], adapterMacAddr)                                                             // Src mac address
	binary.BigEndian.PutUint16(outgoing[ether.EtheroffsetProtocolTypeLength:ether.EtheroffsetProtocolTypeLength+ether.ProtocolTypeLength], ether.EthTypeARP) // Ether protocol type

	header := ARPHeader{
		HardwareType: LinkType(req.AddrType),
		ProtocolType: EthernetType(req.Protocol),
		HardwareSize: uint8(HardwareSize),
		ProtocolSize: uint8(ProtocolSize),
		OpCode:       ARPReply,
	}

	res := &ARP{
		Header:                header,
		SourceHardwareAddress: adapterMacAddr,
		SourceProtocolAddress: req.DstProtAddress,
		TargetHardwareAddress: req.SourceHwAddress,
		TargetProtocolAddress: req.SourceProtAddress,
	}

	outgoing = append(outgoing[:ether.EtherFrameTotalLength], res.Marshal()...)

	return outgoing
}

// Marshal returns binary data.
// RFC826 - ARP Packet Lenght is 28 byte
func (arp *ARP) Marshal() []byte {
	buf := make([]byte, 28)

	// BaseHeader (8 byte)
	binary.BigEndian.PutUint16(buf[0:2], uint16(arp.Header.HardwareType)) // -> HardwareType (2 byte)
	binary.BigEndian.PutUint16(buf[2:4], uint16(arp.Header.ProtocolType)) // -> ProtocolType (2 byte)
	buf[4] = arp.Header.HardwareSize                                      // -> HardwareSize (1 byte)
	buf[5] = arp.Header.ProtocolSize                                      // -> ProtocolSize (1 byte)
	binary.BigEndian.PutUint16(buf[6:8], uint16(arp.Header.OpCode))       // -> OpCode	   (2 byte)

	hal := 6
	pl := 4

	// Payload (20 byte)
	copy(buf[8:8+hal], arp.SourceHardwareAddress)   // SourceHardwareAddress (6 byte)
	copy(buf[14:14+pl], arp.SourceProtocolAddress)  // SourceProtocolAddress (4 byte)
	copy(buf[18:18+hal], arp.TargetHardwareAddress) // TargetHardwareAddress (6 byte)
	copy(buf[24:24+pl], arp.TargetProtocolAddress)  // TargetProtocolAddress (4 byte)

	return buf
}
