package ether

import "net"

// Structure of ether header.
type EtherHeader struct {
	DstMacAddr net.HardwareAddr
	SrcMacAddr net.HardwareAddr
	ProtoType  uint16
}

// Ether frame protocol type.
const (
	EthTypeIPv4                uint16 = 0x0800
	EthTypeARP                 uint16 = 0x0806
	EthTypeRARP                uint16 = 0x8035
	EthTypeVMTP                uint16 = 0x805b
	EthTypeAppleTalk           uint16 = 0x809b
	EthTypeAARP                uint16 = 0x80f3
	EthTypeIPX                 uint16 = 0x8137
	EthTypeSNMP                uint16 = 0x814c
	EthTypeNetBIOS             uint16 = 0x8191
	EthTypeXTP                 uint16 = 0x817d
	EthTypeIPv6                uint16 = 0x86dd
	EthTypePPPoEDiscoveryStage uint16 = 0x8863
	EthTypePPPoESessionStage   uint16 = 0x8864
	EthTypeRRCP                uint16 = 0x8899
	EthTypeLoopDetection       uint16 = 0x9000
)

// Ether frame length of each field.
const (
	DstMacLength          = 6
	SrcMacLength          = 6
	ProtocolTypeLength    = 2
	EtherFrameTotalLength = DstMacLength + SrcMacLength + ProtocolTypeLength
)

// Ether frame offset of protocol layers.
const (
	Etheroffset = 0
	IPoffset    = 14
)

// Ether frame offset length.
const (
	EtheroffsetDst                = 0
	EtheroffsetSrc                = 6
	EtheroffsetProtocolTypeLength = 12
)

const (
	AllNodeMacAddress          = "33:33:00:00:00:01"
	AllRouterMacAddress        = "33:33:00:00:00:02"
	AllSolicitedNodeMACAddress = "33:33:ff:00:00:00"
)
