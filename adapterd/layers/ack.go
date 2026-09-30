// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// Ack is acknowledge of a CYPHONIC packet.
type Ack struct {
	BaseHeader BaseHeader
	HMAC       HMAC
}

// GenerateAck generates Ack packet.
func (ack *Ack) GenerateAck(pathID ID) {
	if err := SerializeBaseHeader(&ack.BaseHeader); err != nil {
		logger.Error(fmt.Errorf("failed to serialize BaseHeader: %w", err))
	}

	SerializeType(&ack.BaseHeader, TypeClassAck)

	ack.BaseHeader.ID = pathID

	// FIXME: HMAC is fixed.
	ack.HMAC = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	// FIXME: calculate packet length.
	ack.BaseHeader.MessageLength = 48

	logger.Debug("Generate ACK", ack)
}

// Marshal returns binary data.
func (ack *Ack) Marshal() ([]byte, error) {
	buf := make([]byte, 0, ack.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, ack.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalAck unmarshals a structure from binary.
func UnmarshalAck(b []byte) (Ack, error) {
	ack := Ack{
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
		HMAC: b[32:48],
	}

	return ack, nil
}
