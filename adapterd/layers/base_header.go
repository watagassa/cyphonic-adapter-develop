// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// BaseHeaderLen lengths (bytes).
const BaseHeaderLen = 32

// BaseHeader is the header of a CYPHONIC packet.
// CYPHONIC expects to send multiple signaling messages at one time.
// Example: RegistrationRequest and NotificationRequest send at one time.
// Count the number of signaling messages and add to the BaseHeader struct's Count.
type BaseHeader struct {
	TransactionID  uint32
	Version        uint8
	Flag           uint8
	Type           uint8
	Count          uint8
	SequenceNumber uint32
	MessageLength  uint16
	NextOpt        uint8
	Reserved       uint8
	ID             ID
}

// ID is a CYPHONIC ID, a slice of bytes.
// length of the byte slice: a 16-byte slice.
type ID []byte

// IDlen lengths (bytes).
const IDlen = 16

// TypeClass defines the class associated with a CYPHONIC packet type.
// classes can be thought of as an array of parallel namespace trees.
type TypeClass uint8

// TypeClass known values.
const (
	TypeClassDefault                    TypeClass = 0
	TypeClassKeySharing                 TypeClass = 1
	TypeClassAck                        TypeClass = 2
	TypeClassLoginRequest               TypeClass = 3
	TypeClassLoginResponse              TypeClass = 4
	TypeClassKeyDistribution            TypeClass = 5
	TypeClassRegistrationRequest        TypeClass = 6
	TypeClassRegistrationResponse       TypeClass = 7
	TypeClassKeepAlive                  TypeClass = 8
	TypeClassNotificationDirection      TypeClass = 9
	TypeClassNotificationRegistration   TypeClass = 10
	TypeClassDirectionRequest           TypeClass = 11
	TypeClassRouteDirectionToCN         TypeClass = 12
	TypeClassRouteDirectionConfirmation TypeClass = 13
	TypeClassRouteDirectionToMN         TypeClass = 14
	TypeClassNodeInformationRequest     TypeClass = 15
	TypeClassNodeInformationResponse    TypeClass = 16
	TypeClassRelayRequest               TypeClass = 17
	TypeClassRelayResponse              TypeClass = 18
	TypeClassNotificationRequest        TypeClass = 19
	TypeClassRouteRequest               TypeClass = 20
	TypeClassHolePunching               TypeClass = 21
	TypeClassTunnelRequest              TypeClass = 22
	TypeClassTunnelResponse             TypeClass = 23
	TypeClassCapsuleMessage             TypeClass = 24
	TypeClassKeepAliveAck               TypeClass = 25
	TypeClassMultiCastRequest           TypeClass = 26
)

// FlagClass defines the class associated with a CYPHONIC packet flag.
// classes can be thought of as an array of parallel namespace trees.
type FlagClass uint8

// FlagClass known values.
const (
	FlagClassDefault                     FlagClass = 0
	FlagClassUnclearErrorUnknown         FlagClass = 1
	FlagClassNoSuchName                  FlagClass = 2
	FlagClassNGFQDN                      FlagClass = 3
	FlagClassDatabaseNoRecord            FlagClass = 4
	FlagClassNoMatchPassword             FlagClass = 5
	FlagClassReplyTimeToLiveUpdateFailed FlagClass = 6
	FlagClassTableNoEntry                FlagClass = 7
	FlagClassNodeTypeMN                  FlagClass = 8
	FlagClassNodeTypeCN                  FlagClass = 9
	FlagClassFirstRegistration           FlagClass = 10
	FlagClassUseVirtualIPv4              FlagClass = 11
	FlagClassOptimization                FlagClass = 12
)

// SerializeBaseHeader is to serialize BaseHeader.
// To set CYPHONIC packet type and flag and etc...
func SerializeBaseHeader(b *BaseHeader) error {
	SerializeVersion(b)
	tidErr := SerializeTransactionID(b)

	if tidErr != nil {
		return fmt.Errorf("Failed to serialize TransactionID: %w", tidErr)
	}

	snErr := SerializeSequenceNumber(b)

	if snErr != nil {
		return fmt.Errorf("Failed to serialize SequenceNumber: %w", snErr)
	}

	idErr := GenerateID(b)

	if idErr != nil {
		return fmt.Errorf("Failed to serialize ID: %w", idErr)
	}

	return nil
}

// SerializeVersion is to serialize BaseHeader's version.
func SerializeVersion(base *BaseHeader) (v uint8) {
	v = 1
	base.Version = v

	return v
}

// SerializeType is to serialize BaseHeader's type.
func SerializeType(base *BaseHeader, tClass TypeClass) (v uint8) {
	base.Type = uint8(tClass)

	return v
}

// SerializeFlag is to serialize BaseHeader's flag.
func SerializeFlag(base *BaseHeader, fClass FlagClass) (v uint8) {
	base.Flag = uint8(fClass)

	return v
}

// SerializeTransactionID is to serialize BaseHeader's TransactionID.
func SerializeTransactionID(base *BaseHeader) error {
	tid, err := GenerateTransactionID()
	if err != nil {
		return err
	}

	base.TransactionID = tid

	return nil
}

// SerializeSequenceNumber is to serialize BaseHeader's SequenceNumber.
func SerializeSequenceNumber(base *BaseHeader) error {
	sn, err := GenerateSequenceNumber()
	if err != nil {
		return err
	}

	base.SequenceNumber = sn

	return nil
}

// GenerateTransactionID is to generate TransactionID.
// TransactionID is 32bit length(4 Bytes).
func GenerateTransactionID() (t uint32, err error) {
	err = binary.Read(rand.Reader, binary.BigEndian, &t)

	if err != nil {
		return 0, err
	}

	return t, err
}

// GenerateSequenceNumber is to generate SequenceNumber.
// SequenceNumber is 32bit length(4 Bytes).
func GenerateSequenceNumber() (t uint32, err error) {
	err = binary.Read(rand.Reader, binary.BigEndian, &t)

	if err != nil {
		return 0, err
	}

	return t, err
}

// GenerateID is to generate ID.
// ID is 128bit length (16 Bytes).
func GenerateID(base *BaseHeader) error {
	base.ID = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	// uuid, err := uuid.NewUUID()
	// if err != nil {
	// 	return err
	// }

	// base.ID, err = uuid.MarshalBinary()
	// if err != nil {
	// 	return err
	// }

	return nil
}
