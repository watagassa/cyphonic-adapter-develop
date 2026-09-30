// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// CapsuleMessage is Capsule Message of a CYPHONIC packet.
type CapsuleMessage struct {
	BaseHeader    BaseHeader
	PayloadLength uint16
	Padding       uint16
	Payload       []byte
	HMAC          HMAC
}

// Marshal returns binary data.
func (cm *CapsuleMessage) Marshal(key []byte) ([]byte, error) {
	planeBuf := make([]byte, 0, cm.BaseHeader.MessageLength-32-16)
	planeBuffer := bytes.NewBuffer(planeBuf)

	if err := binary.Write(planeBuffer, binary.BigEndian, cm.PayloadLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, cm.Padding); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, cm.Payload); err != nil {
		return nil, err
	}

	chiperBuf, err := EncryptPacket(planeBuffer.Bytes(), key)
	if err != nil {
		return nil, err
	}

	cm.BaseHeader.MessageLength = 32 + uint16(len(chiperBuf)) + 16

	buf := make([]byte, 0, cm.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, cm.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, chiperBuf); err != nil {
		return nil, err
	}

	cm.HMAC = GenerateHMAC(buffer.Bytes(), GenerateSalt(&cm.BaseHeader))

	if err := binary.Write(buffer, binary.BigEndian, cm.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalCapsuleMessage unmarshals a structure from binary.
func UnmarshalCapsuleMessage(b []byte, key string) (CapsuleMessage, error) {
	encryptedLen := binary.BigEndian.Uint16(b[12:14]) - HMACLen
	pb, err := DecryptPacket(b[32:encryptedLen], []byte(key))
	if err != nil {
		return CapsuleMessage{}, fmt.Errorf("failed to decrypt Capsule Message: %w", err)
	}

	payloadLen := 4 + binary.BigEndian.Uint16(pb[0:2])

	cm := CapsuleMessage{
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
		PayloadLength: binary.BigEndian.Uint16(pb[0:2]),
		Padding:       binary.BigEndian.Uint16(pb[2:4]),
		Payload:       pb[4:payloadLen],
		HMAC:          b[encryptedLen : encryptedLen+HMACLen],
	}

	if ok := DecodeHMAC(b[encryptedLen:encryptedLen+HMACLen], b[0:encryptedLen], GenerateSalt(&cm.BaseHeader)); !ok {
		err := errors.New("failed to HMAC verification")
		return cm, err
	}

	return cm, nil
}
