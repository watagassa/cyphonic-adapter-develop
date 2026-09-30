// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"encoding/binary"
	"errors"
	"net/netip"
)

// LoginResponse is Login Response of a CYPHONIC packet.
// CipherType sets the algorithm used when encrypting and decrypting.
type LoginResponse struct {
	BaseHeader         BaseHeader
	CipherType         TypeCipherClass
	CommonKeyLength    uint16
	ExpireDate         ExpireDate
	FQDNLength         uint16
	Padding            uint16
	VirtualIPv4Address netip.Addr
	VirtualIPv6Address netip.Addr
	NMSIPv4Address     netip.Addr
	NMSIPv6Address     netip.Addr
	CommonKey          []byte
	FQDN               []byte
}

// ExpireDate handles Common Key expiration.
type ExpireDate struct {
	Year  uint16
	Month uint8
	Day   uint8
}

const (
	// YearSize is length of year (bytes).
	YearSize = 2

	// MonthSize is length of month (bytes).
	MonthSize = 1

	// DaySize is length of day (bytes).
	DaySize = 1
)

// Payload length in LoginResponse.
const (
	// CipherTypeSize lengths (bytes).
	CipherTypeSize = 2

	// CommonKeyLengthSize lengths (bytes).
	CommonKeyLengthSize = 2

	// ExpireDateSize lengths (bytes).
	ExpireDateSize = YearSize + MonthSize + DaySize

	// FQDNLengthSize lengths (bytes).
	FQDNLengthSize = 2

	// PaddingSize lengths (bytes).
	LoginResponsePaddingSize = 2

	// VirtualIPv4AddressSize lengths (bytes).
	VirtualIPv4AddressSize = 4

	// VirtualIPv6AddressSize lengths (bytes).
	VirtualIPv6AddressSize = 16

	// NMSIPv4AddressSize lengths (bytes).
	NMSIPv4AddressSize = 4

	// NMSIPv6AddressSize lengths (bytes).
	NMSIPv6AddressSize = 16
)

// BaseLoginResponseLength is length of (base) body part of LoginResponse.
// Also, since the length of FQDN and CommonKey is variable length, when
// marshaling, calculate and add.
const BaseLoginResponseLength = BaseHeaderLen + CipherTypeSize + CommonKeyLengthSize +
	ExpireDateSize + FQDNLengthSize + LoginResponsePaddingSize +
	VirtualIPv4AddressSize + VirtualIPv6AddressSize + NMSIPv4AddressSize + NMSIPv6AddressSize

// An IP is a single IP address, a slice of bytes.
// Functions in this package accept either 4-byte (IPv4)
// or 16-byte (IPv6) slices as input.
//
// Note that in this documentation, referring to an
// IP address as an IPv4 address or an IPv6 address
// is a semantic property of the address, not just the
// length of the byte slice: a 16-byte slice can still
// be an IPv4 address.
type IP []byte

// TypeCipherClass defines the class associated with crypt type in CYPHONIC packets.
type TypeCipherClass uint16

// TypeCipherClass known values.
const (
	AES256CBC TypeCipherClass = 0
	AES256CFB TypeCipherClass = 1
	AES256OFB TypeCipherClass = 2
	AES256CTR TypeCipherClass = 3
	AES256GCM TypeCipherClass = 4
)

// IsLoginResponse determines if the packet is a LoginResponse packet.
func IsLoginResponse(b []byte) bool {
	if len(b) == 0 {
		return false
	}

	return b[6] == uint8(TypeClassLoginResponse)
}

// UnmarshalLoginResponse unmarshals a structure from binary.
func UnmarshalLoginResponse(b []byte) (LoginResponse, error) {
	virtualIPv4addr, ok := netip.AddrFromSlice(b[44:48])
	if !ok {
		err := errors.New("failed to parse to netip addr type")
		return LoginResponse{}, err
	}

	virtualIPv6addr, ok := netip.AddrFromSlice(b[48:64])
	if !ok {
		err := errors.New("failed to parse to netip addr type")
		return LoginResponse{}, err
	}

	nmsIPv4addr, ok := netip.AddrFromSlice(b[64:68])
	if !ok {
		err := errors.New("failed to parse to netip addr type")
		return LoginResponse{}, err
	}

	nmsIPv6addr, ok := netip.AddrFromSlice(b[68:84])
	if !ok {
		err := errors.New("failed to parse to netip addr type")
		return LoginResponse{}, err
	}

	commonKeyLen := BaseLoginResponseLength + binary.BigEndian.Uint16(b[34:36])
	fqdnLen := BaseLoginResponseLength + binary.BigEndian.Uint16(b[34:36]) + binary.BigEndian.Uint16(b[40:42])

	p := LoginResponse{
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
		CipherType:      TypeCipherClass(binary.BigEndian.Uint16(b[32:34])),
		CommonKeyLength: binary.BigEndian.Uint16(b[34:36]),
		ExpireDate: ExpireDate{
			Year:  binary.BigEndian.Uint16(b[36:38]),
			Month: b[38],
			Day:   b[39],
		},
		FQDNLength:         binary.BigEndian.Uint16(b[40:42]),
		Padding:            binary.BigEndian.Uint16(b[42:44]),
		VirtualIPv4Address: virtualIPv4addr,
		VirtualIPv6Address: virtualIPv6addr,
		NMSIPv4Address:     nmsIPv4addr,
		NMSIPv6Address:     nmsIPv6addr,
		CommonKey:          b[84:commonKeyLen],
		FQDN:               b[commonKeyLen:fqdnLen],
	}

	return p, nil
}
