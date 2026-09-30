// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// RegistrationResponse is Registration Response of a CYPHONIC packet.
type RegistrationResponse struct {
	BaseHeader BaseHeader
	NATFlag    NATFlagClass
	Padding    [3]byte
	HMAC       HMAC
}

// NATFlagClass defines the class associated with NAT.
// classes can be thought of as an array of parallel namespace trees.
type NATFlagClass uint8

// NATFlagClass knowns values.
const (
	TypeNATFlagNone NATFlagClass = 0
	TypeNATFlagNAT  NATFlagClass = 1
)

// Payload length in RegistrationResponse
const (
	// NATFlagSize lengths (bytes).
	NATFlagSize = 1

	// PaddingSize lengths (bytes).
	RegistrationResponsePaddingSize = 3
)

// BaseRegistrationResponseLength is length of (base) body part of RegistrationResponse.
const BaseRegistrationResponseLength = BaseHeaderLen + NATFlagSize + RegistrationResponsePaddingSize

// UnmarshalRegistrationResponse unmarshals a structure from binary.
func UnmarshalRegistrationResponse(b, key []byte) (RegistrationResponse, error) {
	encryptedLen := binary.BigEndian.Uint16(b[12:14]) - HMACLen
	pb, err := DecryptPacket(b[BaseHeaderLen:encryptedLen], key)
	if err != nil {
		return RegistrationResponse{}, fmt.Errorf("failed to decrypt Registration Response: %w", err)
	}

	salt := InitSalt(b)

	if ok := DecodeHMAC(b[encryptedLen:encryptedLen+HMACLen], b[0:encryptedLen], salt); !ok {
		err := errors.New("decode HMAC error")
		return RegistrationResponse{}, err
	}

	rr := RegistrationResponse{
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
		NATFlag: NATFlagClass(pb[0]),
		Padding: [3]byte{0, 0, 0},
		HMAC:    b[encryptedLen : encryptedLen+HMACLen],
	}

	return rr, nil
}
