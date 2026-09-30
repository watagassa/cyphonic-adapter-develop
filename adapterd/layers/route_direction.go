// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
)

// RouteDirection is Route Direction of a CYPHONIC packet.
type RouteDirection struct {
	BaseHeader          BaseHeader
	PathID              ID
	ProcessCode         ProcessCodeClass
	ExecMode            uint8
	GeneralNodeFlag     GeneralNodeFlagClass
	NATmnPort           uint16
	NATcnPort           uint16
	NATmnIPv4           netip.Addr
	NATmnIPv6           netip.Addr
	NATcnIPv4           netip.Addr
	NATcnIPv6           netip.Addr
	MNRealIPv4          netip.Addr
	MNRealIPv6          netip.Addr
	CNRealIPv4          netip.Addr
	CNRealIPv6          netip.Addr
	MNVirtualIPv4       netip.Addr
	MNVirtualIPv6       netip.Addr
	CNVirtualIPv4       netip.Addr
	CNVirtualIPv6       netip.Addr
	TRSIPv4             netip.Addr
	TRSIPv6             netip.Addr
	TunnelKeyCipherType uint16
	TunnelKeyLength     uint16
	TemporaryKeyLength  uint16
	Padding             [2]byte
	ExpireDate          ExpireDate
	FQDNmnLength        uint16
	FQDNcnLength        uint16
	TunnelKey           []byte
	TemporaryKey        []byte
	MNFQDN              []byte
	CNFQDN              []byte
	HMAC                HMAC
}

// ProcessCodeClass defines the class process code type.
type ProcessCodeClass uint8

// ProcessCodeClass known values.
const (
	TunnelRequestToCN    ProcessCodeClass = 1
	TunnelRequestToNATcn ProcessCodeClass = 2
	TunnelRequestToTRS   ProcessCodeClass = 3
)

// GeneralNodeFlagClass defines the class General Node Flag.
type GeneralNodeFlagClass uint16

// GeneralNodeFlagClass known values.
const (
	CYPHONICNode GeneralNodeFlagClass = 0
	GeneralNode  GeneralNodeFlagClass = 1
)

const (
	untilPasswordLengthSize = 56
	untilFQDNLengthSize     = 180
)

// ChangeRouteDirectionType changes route direction type.
// it serializes Route Direction Type.
func (rd *RouteDirection) ChangeRouteDirectionType(typ TypeClass, nodeID []byte) {
	SerializeType(&rd.BaseHeader, typ)
	rd.BaseHeader.ID = nodeID
	rd.BaseHeader.SequenceNumber++
}

// HasSameNAT return true or false.
func (rd *RouteDirection) HasSameNAT(ipVersion TypeLocalIPVersion) bool {
	switch ipVersion {
	case TypeLocalIPVersion4:
		return (rd.NATmnIPv4.Unmap() == rd.NATcnIPv4.Unmap())
	case TypeLocalIPVersion6, TypeLocalDualStackNetwork:
		return (rd.NATmnIPv6 == rd.NATcnIPv6)
	}

	return false
}

// Marshal returns binary data.
func (rd *RouteDirection) Marshal(key []byte) ([]byte, error) {
	planeBuf := make([]byte, 0, rd.BaseHeader.MessageLength-BaseHeaderLen-HMACLen)
	planeBuffer := bytes.NewBuffer(planeBuf)

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.PathID); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.ProcessCode); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.ExecMode); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.GeneralNodeFlag); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.NATmnPort); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.NATcnPort); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.NATmnIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.NATmnIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.NATcnIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.NATcnIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.MNRealIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.MNRealIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.CNRealIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.CNRealIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.MNVirtualIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.MNVirtualIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.CNVirtualIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.CNVirtualIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TRSIPv4.As4()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TRSIPv6.As16()); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TunnelKeyCipherType); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TunnelKeyLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TemporaryKeyLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.Padding); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.ExpireDate); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.FQDNmnLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.FQDNcnLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TunnelKey); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.TemporaryKey); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.MNFQDN); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, rd.CNFQDN); err != nil {
		return nil, err
	}

	chiperBuf, err := EncryptPacket(planeBuffer.Bytes(), key)
	if err != nil {
		return nil, err
	}

	rd.BaseHeader.MessageLength = BaseHeaderLen + uint16(len(chiperBuf)) + HMACLen

	buf := make([]byte, 0, rd.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, rd.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, chiperBuf); err != nil {
		return nil, err
	}

	rd.HMAC = GenerateHMAC(buffer.Bytes(), GenerateSalt(&rd.BaseHeader))

	if err := binary.Write(buffer, binary.BigEndian, rd.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalRouteDirection unmarshals a structure from binary.
func UnmarshalRouteDirection(b, key []byte) (RouteDirection, error) {
	encryptedLen := binary.BigEndian.Uint16(b[12:14]) - HMACLen
	pb, err := DecryptPacket(b[BaseHeaderLen:encryptedLen], key)
	if err != nil {
		return RouteDirection{}, fmt.Errorf("failed to decrypt Route Direction : %w", err)
	}

	salt := InitSalt(b)

	if ok := DecodeHMAC(b[encryptedLen:encryptedLen+HMACLen], b[0:encryptedLen], salt); !ok {
		err := errors.New("decode HMAC error")
		return RouteDirection{}, err
	}

	tunKeyLen := untilFQDNLengthSize + binary.BigEndian.Uint16(pb[166:168])
	tempKeyLen := tunKeyLen + binary.BigEndian.Uint16(pb[168:170])
	fqdnMNLen := tempKeyLen + binary.BigEndian.Uint16(pb[176:178])
	fqdnCNLen := fqdnMNLen + binary.BigEndian.Uint16(pb[178:180])

	natMNIPv4, _ := netip.AddrFromSlice(pb[24:28])
	natMNIPv6, _ := netip.AddrFromSlice(pb[28:44])
	natCNIPv4, _ := netip.AddrFromSlice(pb[44:48])
	natCNIPv6, _ := netip.AddrFromSlice(pb[48:64])
	mnRealIPv4, _ := netip.AddrFromSlice(pb[64:68])
	mnRealIPv6, _ := netip.AddrFromSlice(pb[68:84])
	cnRealIPv4, _ := netip.AddrFromSlice(pb[84:88])
	cnRealIPv6, _ := netip.AddrFromSlice(pb[88:104])
	mnVirtualIPv4, _ := netip.AddrFromSlice(pb[104:108])
	mnVirtualIPv6, _ := netip.AddrFromSlice(pb[108:124])
	cnVirtualIPv4, _ := netip.AddrFromSlice(pb[124:128])
	cnVirtualIPv6, _ := netip.AddrFromSlice(pb[128:144])
	trsIPv4, _ := netip.AddrFromSlice(pb[144:148])
	trsIPv6, _ := netip.AddrFromSlice(pb[148:164])

	rd := RouteDirection{
		BaseHeader: BaseHeader{
			TransactionID:  binary.BigEndian.Uint32(b[0:4]),
			Version:        b[4],
			Flag:           b[5],
			Type:           b[6],
			Count:          b[7],
			SequenceNumber: binary.BigEndian.Uint32(b[8:12]),
			MessageLength:  binary.BigEndian.Uint16(b[12:14]),
			NextOpt:        b[14],
			Reserved:       b[15],
			ID:             b[16:32],
		},
		PathID:              pb[0:16],
		ProcessCode:         ProcessCodeClass(pb[16]),
		ExecMode:            pb[17],
		GeneralNodeFlag:     GeneralNodeFlagClass(binary.BigEndian.Uint16(pb[18:20])),
		NATmnPort:           binary.BigEndian.Uint16(pb[20:22]),
		NATcnPort:           binary.BigEndian.Uint16(pb[22:24]),
		NATmnIPv4:           natMNIPv4,
		NATmnIPv6:           natMNIPv6,
		NATcnIPv4:           natCNIPv4,
		NATcnIPv6:           natCNIPv6,
		MNRealIPv4:          mnRealIPv4,
		MNRealIPv6:          mnRealIPv6,
		CNRealIPv4:          cnRealIPv4,
		CNRealIPv6:          cnRealIPv6,
		MNVirtualIPv4:       mnVirtualIPv4,
		MNVirtualIPv6:       mnVirtualIPv6,
		CNVirtualIPv4:       cnVirtualIPv4,
		CNVirtualIPv6:       cnVirtualIPv6,
		TRSIPv4:             trsIPv4,
		TRSIPv6:             trsIPv6,
		TunnelKeyCipherType: binary.BigEndian.Uint16(pb[164:166]),
		TunnelKeyLength:     binary.BigEndian.Uint16(pb[166:168]),
		TemporaryKeyLength:  binary.BigEndian.Uint16(pb[168:170]),
		Padding:             [2]byte{0, 0},
		ExpireDate: ExpireDate{
			Year:  binary.BigEndian.Uint16(pb[172:174]),
			Month: pb[174],
			Day:   pb[175],
		},
		FQDNmnLength: binary.BigEndian.Uint16(pb[176:178]),
		FQDNcnLength: binary.BigEndian.Uint16(pb[178:180]),
		TunnelKey:    pb[untilFQDNLengthSize:tunKeyLen],
		TemporaryKey: pb[tunKeyLen:tempKeyLen],
		MNFQDN:       pb[tempKeyLen:fqdnMNLen],
		CNFQDN:       pb[fqdnMNLen:fqdnCNLen],
		HMAC:         b[encryptedLen : encryptedLen+HMACLen],
	}

	return rd, nil
}
